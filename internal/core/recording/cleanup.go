package recording

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/goddd/pkg/system"
	"github.com/ixugo/goddd/pkg/web"
	"gorm.io/gorm"
)

// StartCleanupWorker 启动定时清理协程
// 程序启动时执行一次清理，随后每 60 分钟执行一次
func (c Core) StartCleanupWorker() {
	if c.conf == nil || c.conf.Disabled {
		slog.Info("recording cleanup disabled")
		return
	}

	slog.Info("recording cleanup worker started",
		"retain_days", c.conf.RetainDays,
		"disk_threshold", c.conf.DiskUsageThreshold,
		"storage_dir", c.conf.StorageDir,
	)

	// 程序启动时先执行一次清理
	c.runCleanup()

	// 每 60 分钟执行一次
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.runCleanup()
	}
}

// runCleanup 执行清理流程：先预标记即将过期的录像，再清理过期录像，最后处理磁盘空间
func (c Core) runCleanup() {
	c.markExpiringRecordings()
	c.cleanupExpiredRecordings()
	c.cleanupByDiskUsage()
}

// markExpiringRecordings 预标记 1 小时内即将过期的录像
func (c Core) markExpiringRecordings() {
	if c.conf.RetainDays <= 0 {
		return
	}

	ctx := context.Background()
	// 计算 1 小时后的过期时间阈值
	// 如果录像的 started_at < (now + 1h - retain_days)，则该录像将在 1 小时内过期
	expiryCutoff := time.Now().Add(time.Hour).AddDate(0, 0, -c.conf.RetainDays)

	// 批量更新 delete_flag
	err := c.store.Recording().Session(ctx, func(tx *gorm.DB) error {
		return tx.Model(&Recording{}).
			Where("delete_flag = ?", false).
			Where("started_at < ?", orm.Time{Time: expiryCutoff}).
			Update("delete_flag", true).Error
	})
	if err != nil {
		slog.Warn("failed to mark expiring recordings", "err", err)
	}
}

// cleanupExpiredRecordings 清理超过保留天数的录像
func (c Core) cleanupExpiredRecordings() {
	if c.conf.RetainDays <= 0 {
		return
	}

	ctx := context.Background()
	cutoffTime := time.Now().AddDate(0, 0, -c.conf.RetainDays)

	totalDeleted, filesDeleted, freedBytes, failedFiles := c.batchDeleteRecordings(ctx,
		"expired",
		orm.Where("started_at < ?", orm.Time{Time: cutoffTime}),
	)

	if totalDeleted > 0 || failedFiles > 0 {
		slog.Info("expired recording cleanup completed",
			"reason", "retention_policy",
			"retain_days", c.conf.RetainDays,
			"cutoff_time", cutoffTime.Format(time.DateTime),
			"recordings_deleted", totalDeleted,
			"files_deleted", filesDeleted,
			"failed_files", failedFiles,
			"freed_bytes", freedBytes,
		)
	}
}

// cleanupByDiskUsage 基于磁盘使用率清理录像
// 当磁盘使用率超过阈值时，删除最旧的录像直到使用率降到阈值以下
// 同时预标记未来 2 小时可能被删除的文件
func (c Core) cleanupByDiskUsage() {
	if c.conf.DiskUsageThreshold <= 0 || c.conf.DiskUsageThreshold >= 100 {
		return
	}

	storageDir := c.conf.StorageDir
	if storageDir == "" {
		storageDir = "./recordings"
	}

	absStorageDir := filepath.Join(system.Getwd(), storageDir)
	if _, err := os.Stat(absStorageDir); os.IsNotExist(err) {
		return
	}

	usage, err := getDiskUsage(absStorageDir)
	if err != nil {
		slog.Warn("failed to get disk usage", "err", err)
		return
	}

	if usage < c.conf.DiskUsageThreshold {
		return
	}

	ctx := context.Background()

	// 计算过去一小时的录像总大小，作为需要清理的目标
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	var recentRecordings []*Recording
	_, _ = c.store.Recording().Find(ctx, &recentRecordings, nil,
		orm.Where("created_at >= ?", orm.Time{Time: oneHourAgo}),
	)

	var recentSize int64
	for _, r := range recentRecordings {
		recentSize += r.Size
	}
	// 至少清理 100MB
	if recentSize < 100*1024*1024 {
		recentSize = 100 * 1024 * 1024
	}

	// 删除最旧的录像
	var freedBytes int64
	var deletedCount, failedCount int
	batchSize := 50

	for freedBytes < recentSize {
		var oldestRecordings []*Recording
		pager := web.PagerFilter{Page: 1, Size: batchSize}
		_, err := c.store.Recording().Find(ctx, &oldestRecordings, &pager,
			orm.OrderBy("started_at ASC"),
		)
		if err != nil || len(oldestRecordings) == 0 {
			break
		}

		// 收集待删除的文件路径和 ID
		var deleteIDs []int64
		var batchFreed int64
		var batchFailed int

		for _, rec := range oldestRecordings {
			filePath := rec.Path
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(system.Getwd(), filePath)
			}
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				batchFailed++
			} else {
				batchFreed += rec.Size
			}
			deleteIDs = append(deleteIDs, rec.ID)
		}

		// 批量删除数据库记录
		if len(deleteIDs) > 0 {
			_ = c.store.Recording().Session(ctx, func(tx *gorm.DB) error {
				return tx.Where("id IN ?", deleteIDs).Delete(&Recording{}).Error
			})
			deletedCount += len(deleteIDs)
		}

		freedBytes += batchFreed
		failedCount += batchFailed

		// 检查磁盘使用率
		usage, err = getDiskUsage(absStorageDir)
		if err == nil && usage < c.conf.DiskUsageThreshold {
			break
		}
	}

	// 清理空目录
	cleanupEmptyDirs(absStorageDir)

	// 预标记未来 2 小时可能被删除的文件
	c.markNextDeletionCandidates(ctx, freedBytes*2)

	if deletedCount > 0 || failedCount > 0 {
		slog.Info("disk usage cleanup completed",
			"reason", "disk_threshold_exceeded",
			"initial_usage", usage,
			"threshold", c.conf.DiskUsageThreshold,
			"recordings_deleted", deletedCount,
			"failed_files", failedCount,
			"freed_bytes", freedBytes,
		)
	}
}

// markNextDeletionCandidates 预标记即将被删除的录像
// 标记最旧的、总大小约等于 targetSize 的录像为待删除状态
func (c Core) markNextDeletionCandidates(ctx context.Context, targetSize int64) {
	if targetSize <= 0 {
		return
	}

	// 查询未被标记的最旧录像
	var candidates []*Recording
	pager := web.PagerFilter{Page: 1, Size: 200}
	_, err := c.store.Recording().Find(ctx, &candidates, &pager,
		orm.Where("delete_flag = ?", false),
		orm.OrderBy("started_at ASC"),
	)
	if err != nil || len(candidates) == 0 {
		return
	}

	// 计算需要标记的录像
	var markedSize int64
	var markIDs []int64
	for _, rec := range candidates {
		if markedSize >= targetSize {
			break
		}
		markIDs = append(markIDs, rec.ID)
		markedSize += rec.Size
	}

	// 批量更新标记
	if len(markIDs) > 0 {
		_ = c.store.Recording().Session(ctx, func(tx *gorm.DB) error {
			return tx.Model(&Recording{}).Where("id IN ?", markIDs).Update("delete_flag", true).Error
		})
	}
}

// batchDeleteRecordings 批量删除录像（文件+数据库记录）
// reason 参数用于日志记录，说明删除原因
func (c Core) batchDeleteRecordings(ctx context.Context, reason string, conditions ...orm.QueryOption) (totalDeleted, filesDeleted, failedFiles int, freedBytes int64) {
	batchSize := 100

	for {
		var recordings []*Recording
		pager := web.PagerFilter{Page: 1, Size: batchSize}
		_, err := c.store.Recording().Find(ctx, &recordings, &pager, conditions...)
		if err != nil || len(recordings) == 0 {
			break
		}

		// 收集删除结果
		var deleteIDs []int64
		var batchFreed int64
		var batchFilesDeleted, batchFailed int

		for _, rec := range recordings {
			filePath := rec.Path
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(system.Getwd(), filePath)
			}
			if err := os.Remove(filePath); err != nil {
				if !os.IsNotExist(err) {
					batchFailed++
				}
			} else {
				batchFilesDeleted++
				batchFreed += rec.Size
			}
			deleteIDs = append(deleteIDs, rec.ID)
		}

		// 批量删除数据库记录
		if len(deleteIDs) > 0 {
			err = c.store.Recording().Session(ctx, func(tx *gorm.DB) error {
				return tx.Where("id IN ?", deleteIDs).Delete(&Recording{}).Error
			})
			if err == nil {
				totalDeleted += len(deleteIDs)
			}
		}

		filesDeleted += batchFilesDeleted
		failedFiles += batchFailed
		freedBytes += batchFreed
	}

	// 清理空目录
	if c.conf != nil && c.conf.StorageDir != "" {
		absStorageDir := filepath.Join(system.Getwd(), c.conf.StorageDir)
		cleanupEmptyDirs(absStorageDir)
	}

	return
}

// getDiskUsage 获取指定路径所在磁盘的使用率（百分比）
func getDiskUsage(path string) (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0, nil
	}

	usage := float64(used) / float64(total) * 100
	return usage, nil
}

// cleanupEmptyDirs 递归删除空目录
func cleanupEmptyDirs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dir, entry.Name())
			cleanupEmptyDirs(subDir)

			// 检查子目录是否为空
			subEntries, err := os.ReadDir(subDir)
			if err == nil && len(subEntries) == 0 {
				_ = os.Remove(subDir)
			}
		}
	}
}
