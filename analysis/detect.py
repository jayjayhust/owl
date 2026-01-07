import logging
import time
from typing import Any

import numpy as np
import cv2
import onnxruntime as ort


slog = logging.getLogger("Detector")

# COCO 数据集 80 类标签
COCO_LABELS = [
    "person",
    "bicycle",
    "car",
    "motorcycle",
    "airplane",
    "bus",
    "train",
    "truck",
    "boat",
    "traffic light",
    "fire hydrant",
    "stop sign",
    "parking meter",
    "bench",
    "bird",
    "cat",
    "dog",
    "horse",
    "sheep",
    "cow",
    "elephant",
    "bear",
    "zebra",
    "giraffe",
    "backpack",
    "umbrella",
    "handbag",
    "tie",
    "suitcase",
    "frisbee",
    "skis",
    "snowboard",
    "sports ball",
    "kite",
    "baseball bat",
    "baseball glove",
    "skateboard",
    "surfboard",
    "tennis racket",
    "bottle",
    "wine glass",
    "cup",
    "fork",
    "knife",
    "spoon",
    "bowl",
    "banana",
    "apple",
    "sandwich",
    "orange",
    "broccoli",
    "carrot",
    "hot dog",
    "pizza",
    "donut",
    "cake",
    "chair",
    "couch",
    "potted plant",
    "bed",
    "dining table",
    "toilet",
    "tv",
    "laptop",
    "mouse",
    "remote",
    "keyboard",
    "cell phone",
    "microwave",
    "oven",
    "toaster",
    "sink",
    "refrigerator",
    "book",
    "clock",
    "vase",
    "scissors",
    "teddy bear",
    "hair drier",
    "toothbrush",
]


class ObjectDetector:

    def __init__(self, model_path: str = "yolo11n.onnx"):
        self.model_path = model_path
        self.session: ort.InferenceSession | None = None
        self.input_name: str = ""
        self.input_shape: tuple = (1, 3, 640, 640)
        self._is_ready = False
        self.names: dict[int, str] = {i: name for i, name in enumerate(COCO_LABELS)}

    def load_model(self) -> bool:
        """加载 ONNX 模型并初始化推理会话"""
        try:
            slog.info(f"加载 ONNX 模型: {self.model_path} ...")
            start_time = time.time()

            # 配置 ONNX Runtime 会话选项
            sess_options = ort.SessionOptions()
            sess_options.graph_optimization_level = (
                ort.GraphOptimizationLevel.ORT_ENABLE_ALL
            )
            # 限制线程数，避免在容器中占用过多 CPU
            sess_options.intra_op_num_threads = 4
            sess_options.inter_op_num_threads = 2

            # 优先使用 CPU 执行提供程序
            providers = ["CPUExecutionProvider"]

            self.session = ort.InferenceSession(
                self.model_path, sess_options=sess_options, providers=providers
            )

            # 获取输入信息
            input_info = self.session.get_inputs()[0]
            self.input_name = input_info.name
            self.input_shape = tuple(input_info.shape)

            # 预热模型
            dummy_img = np.zeros((640, 640, 3), dtype=np.uint8)
            self._preprocess(dummy_img)
            dummy_input = self._preprocess(dummy_img)
            self.session.run(None, {self.input_name: dummy_input})

            elapsed = time.time() - start_time
            slog.info(
                f"ONNX 模型加载完成 (耗时: {elapsed:.2f}s, 输入形状: {self.input_shape})"
            )
            self._is_ready = True
            return True
        except Exception as e:
            slog.error(f"加载 ONNX 模型失败: {e}")
            return False

    def is_ready(self) -> bool:
        return self._is_ready and self.session is not None

    def _preprocess(self, image: np.ndarray) -> np.ndarray:
        """
        预处理图像：调整大小、归一化、转换格式
        YOLO 期望输入格式: NCHW, float32, 归一化到 [0, 1]
        """
        # 保持宽高比的 letterbox 缩放
        target_size = self.input_shape[2]  # 640
        h, w = image.shape[:2]
        scale = min(target_size / h, target_size / w)
        new_h, new_w = int(h * scale), int(w * scale)

        # 缩放图像
        resized = cv2.resize(image, (new_w, new_h), interpolation=cv2.INTER_LINEAR)

        # 创建正方形画布并居中放置图像
        canvas = np.full((target_size, target_size, 3), 114, dtype=np.uint8)
        top = (target_size - new_h) // 2
        left = (target_size - new_w) // 2
        canvas[top : top + new_h, left : left + new_w] = resized

        # BGR -> RGB, HWC -> CHW, 归一化
        blob = canvas[:, :, ::-1].transpose(2, 0, 1).astype(np.float32) / 255.0
        return np.expand_dims(blob, axis=0)

    def _postprocess(
        self,
        outputs: np.ndarray,
        original_shape: tuple[int, int],
        threshold: float,
        label_filter: list[str] | None = None,
    ) -> list[dict[str, Any]]:
        """
        后处理 YOLO 输出：解析检测框、应用 NMS、坐标转换
        YOLO11 输出格式: (1, 84, 8400) -> 84 = 4 (bbox) + 80 (classes)
        """
        # 转置为 (8400, 84) 便于处理
        predictions = outputs[0].T  # (8400, 84)

        # 提取边界框和类别分数
        boxes = predictions[:, :4]  # x_center, y_center, width, height
        scores = predictions[:, 4:]  # 80 个类别的分数

        # 获取每个检测框的最高分数和对应类别
        class_ids = np.argmax(scores, axis=1)
        confidences = np.max(scores, axis=1)

        # 过滤低置信度检测
        mask = confidences >= threshold
        boxes = boxes[mask]
        confidences = confidences[mask]
        class_ids = class_ids[mask]

        if len(boxes) == 0:
            return []

        # 转换坐标：center_x, center_y, w, h -> x1, y1, x2, y2
        x_center, y_center, w, h = boxes[:, 0], boxes[:, 1], boxes[:, 2], boxes[:, 3]
        x1 = x_center - w / 2
        y1 = y_center - h / 2
        x2 = x_center + w / 2
        y2 = y_center + h / 2

        # 缩放坐标到原始图像尺寸
        orig_h, orig_w = original_shape
        target_size = self.input_shape[2]
        scale = min(target_size / orig_h, target_size / orig_w)
        pad_h = (target_size - orig_h * scale) / 2
        pad_w = (target_size - orig_w * scale) / 2

        x1 = (x1 - pad_w) / scale
        y1 = (y1 - pad_h) / scale
        x2 = (x2 - pad_w) / scale
        y2 = (y2 - pad_h) / scale

        # 裁剪到图像边界
        x1 = np.clip(x1, 0, orig_w)
        y1 = np.clip(y1, 0, orig_h)
        x2 = np.clip(x2, 0, orig_w)
        y2 = np.clip(y2, 0, orig_h)

        # NMS (非极大值抑制)
        boxes_for_nms = np.stack([x1, y1, x2, y2], axis=1)
        indices = cv2.dnn.NMSBoxes(
            boxes_for_nms.tolist(),
            confidences.tolist(),
            threshold,
            0.45,  # NMS IoU 阈值
        )

        detections = []

        # 处理 NMSBoxes 返回值的不同格式
        # OpenCV 不同版本返回格式不同：可能是 list、tuple、ndarray
        if indices is None or len(indices) == 0:
            return detections

        # 将 indices 转换为一维列表
        if isinstance(indices, np.ndarray):
            indices = indices.flatten().tolist()
        elif isinstance(indices, tuple):
            indices = list(indices)

        for idx in indices:
            # 确保 idx 是整数
            idx = int(idx) if not isinstance(idx, int) else idx

            cls_id = int(class_ids[idx])
            label = self.names.get(cls_id, f"class_{cls_id}")

            # 标签过滤
            if label_filter and label not in label_filter:
                continue

            x_min_val = int(x1[idx])
            y_min_val = int(y1[idx])
            x_max_val = int(x2[idx])
            y_max_val = int(y2[idx])
            area = (x_max_val - x_min_val) * (y_max_val - y_min_val)

            detections.append(
                {
                    "label": label,
                    "confidence": float(confidences[idx]),
                    "box": {
                        "x_min": x_min_val,
                        "y_min": y_min_val,
                        "x_max": x_max_val,
                        "y_max": y_max_val,
                    },
                    "area": area,
                    "norm_box": {
                        "x": (x_min_val + x_max_val) / 2 / orig_w,
                        "y": (y_min_val + y_max_val) / 2 / orig_h,
                        "w": (x_max_val - x_min_val) / orig_w,
                        "h": (y_max_val - y_min_val) / orig_h,
                    },
                }
            )

        return detections

    def detect(
        self,
        image: np.ndarray,
        threshold: float = 0.5,
        label_filter: list[str] | None = None,
        regions: list[tuple[int, int, int, int]] | None = None,
    ) -> tuple[list[dict], float]:
        """执行目标检测"""
        if not self.is_ready():
            raise RuntimeError("模型未加载")

        start_time = time.time()
        detections = []

        if regions and len(regions) > 0:
            for region in regions:
                x_min, y_min, x_max, y_max = region
                h, w = image.shape[:2]
                x_min = max(0, x_min)
                y_min = max(0, y_min)
                x_max = min(w, x_max)
                y_max = min(h, y_max)

                if x_max <= x_min or y_max <= y_min:
                    continue

                cropped = image[y_min:y_max, x_min:x_max]
                if cropped.size == 0:
                    continue

                region_detections = self._detect_single(
                    cropped, threshold, label_filter
                )

                for det in region_detections:
                    det["box"]["x_min"] += x_min
                    det["box"]["y_min"] += y_min
                    det["box"]["x_max"] += x_min
                    det["box"]["y_max"] += y_min
                    detections.append(det)
        else:
            detections = self._detect_single(image, threshold, label_filter)

        inference_time_ms = (time.time() - start_time) * 1000
        return detections, inference_time_ms

    def _detect_single(
        self, image: np.ndarray, threshold: float, label_filter: list[str] | None = None
    ) -> list[dict[str, Any]]:
        """对单张图像执行检测"""
        if not self.session:
            return []

        # 预处理
        input_tensor = self._preprocess(image)

        # 推理
        outputs = self.session.run(None, {self.input_name: input_tensor})

        # 后处理
        original_shape = image.shape[:2]
        output: np.ndarray = np.asarray(outputs[0])
        return self._postprocess(output, original_shape, threshold, label_filter)


class MotionDetector:
    """
    运动检测器 - 基于背景差分法
    用于在目标检测前预筛选有运动的帧，减少不必要的 AI 推理
    """

    def __init__(self):
        self.backgrounds: dict[str, np.ndarray] = {}
        self.motion_threshold = 25
        self.min_contour_area = 500

    def detect(
        self,
        image: np.ndarray,
        camera_name: str,
        roi_points: list[float] | None = None,
    ) -> tuple[list[dict[str, Any]], bool]:
        h, w = image.shape[:2]
        if len(image.shape) == 3:
            gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        else:
            gray = image.copy()

        # 高斯模糊平滑噪点
        gray = cv2.GaussianBlur(gray, (21, 21), 0)

        if camera_name not in self.backgrounds:
            self.backgrounds[camera_name] = gray.astype(np.float32)
            return [], False

        cv2.accumulateWeighted(gray, self.backgrounds[camera_name], 0.1)

        frame_delta = cv2.absdiff(
            gray, cv2.convertScaleAbs(self.backgrounds[camera_name])
        )
        thresh = cv2.threshold(
            frame_delta, self.motion_threshold, 255, cv2.THRESH_BINARY
        )[1]

        # ROI 区域掩码
        if roi_points and len(roi_points) > 0:
            mask = np.zeros((h, w), dtype=np.uint8)
            pts = []
            for i in range(0, len(roi_points), 2):
                pts.append((int(roi_points[i] * w), int(roi_points[i + 1] * h)))
            pts_np = np.array([pts], dtype=np.int32)
            cv2.fillPoly(mask, [pts_np], (255,))  # type: ignore
            thresh = cv2.bitwise_and(thresh, thresh, mask=mask)

        kernel = np.ones((3, 3), np.uint8)
        thresh = cv2.dilate(thresh, kernel, iterations=2)

        contours, _ = cv2.findContours(
            thresh, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
        )

        motion_boxes = []
        for contour in contours:
            if cv2.contourArea(contour) < self.min_contour_area:
                continue
            x, y, cw, ch = cv2.boundingRect(contour)
            motion_boxes.append(
                {"y_min": y, "x_min": x, "y_max": y + ch, "x_max": x + cw}
            )

        has_motion = len(motion_boxes) > 0
        return motion_boxes, has_motion
