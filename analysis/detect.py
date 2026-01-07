import logging
from tabnanny import verbose
import time
from typing import Any

from sympy import true
from ultralytics import YOLO  # type: ignore
import numpy as np
import cv2


slog = logging.getLogger("Detector")


class ObjectDetector:
    def __init__(self, model_path: str = "yolo11n.pt", device: str = "auto"):
        self.model_path = model_path
        self.device = device
        self.model: YOLO | None = None
        self._is_ready = False

    def load_model(self) -> bool:
        try:
            slog.info(f"加载模型: {self.model_path} ...")
            start_time = time.time()
            self.model = YOLO(self.model_path)
            if self.device != "auto":
                self.model.to(self.device)

            # 预热模型
            dummy_img = np.zeros((640, 640, 3), dtype=np.uint8)
            self.model.predict(dummy_img, verbose=False)

            elapsed = time.time() - start_time
            slog.info(f"模型加载完成 (耗时: {elapsed:.2f}s)")
            self._is_ready = True
            return True
        except Exception as e:
            slog.error(f"加载模型失败: {e}")
            return False
        return True

    def is_ready(self) -> bool:
        return self._is_ready and self.model is not None

    def detect(
        self,
        image: np.ndarray,
        threshold: float = 0.5,
        label_filter: list[str] | None = None,
        regions: list[tuple[int, int, int, int]] | None = None,
    ) -> tuple[list[dict], float]:

        if not self.is_ready:
            raise RuntimeError("模型未加载")

        start_time = time.time()
        detections = []

        if regions and len(regions) > 0:
            for region in regions:
                x_min, y_min, x_max, y_max = region
                h, w = image.shape[:2]
                x_min = max(0, x_max)
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
        if not self.model:
            return []
        results = self.model.predict(image, conf=threshold, verbose=False)
        detections = []

        for result in results:
            if result.boxes is None:
                continue
            for box in result.boxes:
                cls_id = int(box.cls[0])
                label = self.model.names[cls_id]
                confidence = float(box.conf[0])

                if label_filter and label not in label_filter:
                    continue
                x1, y1, x2, y2 = box.xyxy[0].tolist()
                x_min, y_min = int(x1), int(y1)
                x_max, y_max = int(x2), int(y2)

                area = (x_max - x_min) * (y_max - y_min)

                detections.append(
                    {
                        "label": label,
                        "confidence": confidence,
                        "box": {
                            "x_min": x_min,
                            "y_min": y_min,
                            "x_max": x_max,
                            "y_max": y_max,
                        },
                        "area": area,
                        "norm_box": {
                            "x": (x1 + x2) / 2 / image.shape[1],
                            "y": (y1 + y2) / 2 / image.shape[0],
                            "w": (x2 - x1) / image.shape[1],
                            "h": (y2 - y1) / image.shape[0],
                        },
                    }
                )
        return detections


class MotionDetector:
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
        # 模糊可以平滑噪点
        # (21,21) 模糊大小，必须是奇数
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

        if roi_points and len(roi_points) > 0:
            mask = np.zeros((h, w), dtype=np.uint8)

            pts = []
            for i in range(0, len(roi_points), 2):
                pts.append((int(roi_points[i] * w), int(roi_points[i + 1] * h)))
            pts_np = np.array([pts], dtype=np.int32)
            cv2.fillPoly(mask, pts_np, 255)
            thresh = cv2.bitwise_and(thresh, mask=mask)

        thresh = cv2.dilate(thresh, None, iterations=2)

        contours, _ = cv2.findContours(
            thresh, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
        )

        motion_boxes = []
        for contour in contours:
            if cv2.contourArea(contour) < self.min_contour_area:
                continue
            x, y, w, h = cv2.boundingRect(contour)
            motion_boxes.append(
                {"y_min": y, "x_min": x, "y_max": y + h, "x_max": x + w}
            )
        has_motion = len(motion_boxes) > 0
        return motion_boxes, has_motion
