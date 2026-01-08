package pushdb

import (
	"context"
	"testing"

	"github.com/gowvp/owl/internal/core/push"
	"github.com/ixugo/goddd/pkg/orm"
)

func TestStreamPushGet(t *testing.T) {
	db, mock, err := generateMockDB()
	if err != nil {
		t.Fatal(err)
	}
	userDB := NewStreamPush(db)

	mock.ExpectQuery(`SELECT \* FROM "stream_pushs" WHERE id=\$1 (.+) LIMIT \$2`).WithArgs("jack", 1)
	var out push.StreamPush
	if err := userDB.Get(context.Background(), &out, orm.Where("id=?", "jack")); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal("ExpectationsWereMet err:", err)
	}
}
