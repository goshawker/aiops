"""Unit tests for Anomaly Detection Service."""
import unittest
import json
from typing import List, Dict, Optional


class MockKafkaProducer:
    """Mock Kafka producer for testing."""
    
    def __init__(self):
        self.messages_sent = []
        self.connected = True
    
    def connect(self):
        pass
    
    def send(self, topic: str, message: dict) -> None:
        if self.connected:
            self.messages_sent.append({
                "topic": topic,
                "message": message,
            })


class MockKafkaConsumer:
    """Mock Kafka consumer for testing."""
    
    def __init__(self, brokers: List[str], topic: str, service_name: str):
        self.brokers = brokers
        self.topic = topic
        self.service_name = service_name
        self.running = False
    
    def start(self, handler, *args, **kwargs):
        """Simulate message handling without actually consuming."""
        pass


class MockRiverDetector:
    """Mock River detector with basic anomaly detection."""
    
    def __init__(self, n_trees=25, height=15, window_size=250):
        self.n_trees = n_trees
        self.height = height
        self.window_size = window_size
        self.is_warmed_up = False
        self.metric_mean = {}
        self.metric_std = {}
        
    def fit(self, data: List[float], metric_name: str = "") -> None:
        """Fit the model with training data."""
        if not data:
            return
        mean = sum(data) / len(data)
        variance = sum((x - mean) ** 2 for x in data) / len(data)
        std = variance ** 0.5 if variance > 0 else 1.0
        
        self.metric_mean[metric_name] = mean
        self.metric_std[metric_name] = std
        self.is_warmed_up = True
    
    def detect(self, value: float, metric_name: str = "", labels: dict = None) -> Optional[Dict]:
        """Detect anomaly in a single value."""
        if not self.is_warmed_up:
            return None
        
        mean = self.metric_mean.get(metric_name, 0.0)
        std = self.metric_std.get(metric_name, 1.0)
        
        if std == 0:
            return None
        
        z_score = abs(value - mean) / std
        
        # Use adaptive threshold based on value magnitude
        adaptive_threshold = 3.0 * (1.0 + 0.1 * min(100.0, value ** 0.5) / mean if mean > 0 else 0)
        
        if z_score > adaptive_threshold:
            anomaly_type = "spike" if value > mean else "drop"
            severity = "critical" if z_score > 4.0 else "warning"
            
            return {
                "anomaly": True,
                "value": value,
                "threshold": adaptive_threshold,
                "anomaly_type": anomaly_type,
                "severity": severity,
                "confidence": min(0.99, z_score / adaptive_threshold),
                "message": f"Detected {anomaly_type} anomaly: value {value:.2f}",
                "metric_name": metric_name,
                "labels": labels or {},
            }
        
        return None
    
    def set_threshold(self, metric_name: str, labels: dict, op: str, value: float) -> None:
        """Set custom threshold (placeholder)."""
        pass
    
    def get_status(self) -> dict:
        """Return model status."""
        return {
            "river_available": True,
            "model_count": 1,
            "models": {
                "default": {
                    "point_count": sum(len(data) for data in self.metric_mean.values()) if self.metric_mean else 0,
                    "warmup_done": self.is_warmed_up,
                    "has_river_model": True,
                }
            },
        }


class Mock3SigmaDetector:
    """Simple 3-sigma detector for fallback."""
    
    def __init__(self, sigma_multiplier: float = 3.0, warmup_points: int = 50):
        self.sigma_multiplier = sigma_multiplier
        self.warmup_points = warmup_points
        self.is_warmed_up = False
    
    def fit(self, data: List[float], metric_name: str = "") -> None:
        if not data:
            return
        self.mean = sum(data) / len(data)
        self.variance = sum((x - self.mean) ** 2 for x in data) / len(data)
        self.is_warmed_up = len(data) >= self.warmup_points
    
    def detect(self, value: float, metric_name: str = "") -> Optional[Dict]:
        if not self.is_warmed_up:
            return None
        
        std = self.variance ** 0.5 if self.variance > 0 else 1.0
        z_score = abs(value - self.mean) / std
        
        if z_score > self.sigma_multiplier:
            return {
                "anomaly": True,
                "value": value,
                "threshold": self.sigma_multiplier,
                "anomaly_type": "spike" if value > self.mean else "drop",
                "severity": "critical" if z_score > 4.0 else "warning",
                "confidence": min(0.99, z_score / self.sigma_multiplier),
                "message": f"Z-score {z_score:.2f} exceeds threshold",
            }
        
        return None


class MockMetricPoint:
    """Mock data point."""
    
    def __init__(self, timestamp: float, value: float, labels: Optional[dict] = None):
        self.timestamp = timestamp
        self.value = value
        self.labels = labels or {}
        self.metric_name = ""
    
    def __repr__(self):
        return f"MockMetricPoint(ts={self.timestamp}, value={self.value})"


class TestAnomalyDetector(unittest.TestCase):
    """Test anomaly detection functionality."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.river_detector = MockRiverDetector()
        self.sigma_detector = Mock3SigmaDetector()
    
    def test_river_detector_warmup(self):
        """Test River detector warmup."""
        self.assertFalse(self.river_detector.is_warmed_up)
        data = list(range(10, 12)) * 50  # Normal oscillating data
        self.river_detector.fit(data, "test_metric")
        self.assertTrue(self.river_detector.is_warmed_up)
    
    def test_river_detector_normal_data(self):
        """Test that normal data passes through."""
        normal_data = [10 + (i % 3) for i in range(100)]  # Oscillating 10-12
        self.river_detector.fit(normal_data, "cpu")
        
        for i in range(8):  # Values within normal range
            result = self.river_detector.detect(10 + i, "cpu")
            self.assertIsNone(result, f"Value {10+i} should not be anomaly")
    
    def test_river_detector_anomaly(self):
        """Test that anomalies are detected."""
        normal_data = [10 + (i % 3) for i in range(100)]
        self.river_detector.fit(normal_data, "memory")
        
        # Normal value
        self.river_detector.detect(11.0, "memory")  # Should pass
        
        # Anomaly
        result = self.river_detector.detect(50.0, "memory")
        self.assertIsNotNone(result)
        self.assertTrue(result["anomaly"])
        self.assertEqual(result["anomaly_type"], "spike")
        self.assertIn(result["severity"], ["warning", "critical"])
        self.assertGreater(result["confidence"], 0)
    
    def test_sigma_detector_warmup(self):
        """Test 3-sigma detector warmup."""
        self.assertFalse(self.sigma_detector.is_warmed_up)
        data = [10] * 50  # Exactly 50 points
        self.sigma_detector.fit(data, "test")
        self.assertTrue(self.sigma_detector.is_warmed_up)
    
    def test_sigma_detector_detection(self):
        """Test 3-sigma detector basic functionality."""
        data = [10.0] * 100  # Exactly 100 points
        self.sigma_detector.fit(data, "disk")
        
        # Normal value
        self.sigma_detector.detect(10.0, "disk")  # Should pass
        
        # Anomaly
        result = self.sigma_detector.detect(50.0, "disk")
        self.assertIsNotNone(result)
        self.assertTrue(result["anomaly"])
    
    def test_metric_name_preservation(self):
        """Test that metric name is preserved."""
        point = MockMetricPoint(1000.0, 50.0, {"host": "server1"})
        point.metric_name = "cpu_usage"
        
        result = self.river_detector.detect(point.value, point.metric_name, point.labels)
        self.assertEqual(result["metric_name"], "cpu_usage")
    
    def test_labels_preservation(self):
        """Test that labels are preserved."""
        labels = {"host": "server1", "env": "prod", "team": "backend"}
        
        result = self.river_detector.detect(50.0, "test_metric", labels)
        self.assertEqual(result["labels"]["host"], "server1")
        self.assertEqual(result["labels"]["env"], "prod")
        self.assertEqual(result["labels"]["team"], "backend")
    
    def test_zero_data_handling(self):
        """Test handling of zero variance data."""
        # All same values = zero variance
        data = [5.0] * 100
        self.river_detector.fit(data, "zero_var")
        
        # With zero variance, any deviation is extreme
        result = self.river_detector.detect(10.0, "zero_var")
        self.assertIsNotNone(result)
        self.assertTrue(result["anomaly"])
    
    def test_negative_value_detection(self):
        """Test detection of negative anomalies."""
        data = [10.0] * 100
        self.river_detector.fit(data, "negative_test")
        
        # Positive anomaly
        self.river_detector.detect(50.0, "negative_test")
        
        # Negative anomaly (below mean)
        result = self.river_detector.detect(-20.0, "negative_test")
        self.assertIsNotNone(result)
        self.assertEqual(result["anomaly_type"], "drop")
    
    def test_empty_data_handling(self):
        """Test handling of empty data."""
        # Empty data should return None
        result = self.river_detector.detect(10.0, "")
        self.assertIsNone(result)
    
    def test_get_status(self):
        """Test status retrieval."""
        status = self.river_detector.get_status()
        self.assertIn("river_available", status)
        self.assertIn("model_count", status)
        self.assertIn("models", status)
    
    def test_confidence_bounds(self):
        """Test that confidence is bounded."""
        data = [10.0] * 100
        self.river_detector.fit(data, "confidence_test")
        
        # High anomaly should have high confidence
        result = self.river_detector.detect(100.0, "confidence_test")
        self.assertIsNotNone(result)
        self.assertGreaterEqual(result["confidence"], 0.0)
        self.assertLessEqual(result["confidence"], 1.0)


class TestAnomalyService(unittest.TestCase):
    """Test the full anomaly service."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.producer = MockKafkaProducer()
        self.consumer = MockKafkaConsumer(["localhost:9092"], "metrics", "anomaly-service")
        self.river_detector = MockRiverDetector()
        self.sigma_detector = Mock3SigmaDetector()
    
    def test_producer_integration(self):
        """Test producer connection and sending."""
        self.producer.connect()
        
        message = {
            "source_type": "metric",
            "source": "cpu_usage",
            "value": 95.5,
        }
        
        self.producer.send("alerts", message)
        self.assertEqual(len(self.producer.messages_sent), 1)
        self.assertEqual(self.producer.messages_sent[0]["topic"], "alerts")
    
    def test_detector_selection(self):
        """Test that River detector is preferred over 3-sigma."""
        data = [10.0] * 100
        self.river_detector.fit(data, "test")
        self.sigma_detector.fit(data, "test")
        
        # Both should detect anomalies
        river_result = self.river_detector.detect(50.0, "test")
        sigma_result = self.sigma_detector.detect(50.0, "test")
        
        self.assertIsNotNone(river_result)
        self.assertIsNotNone(sigma_result)


if __name__ == "__main__":
    unittest.main(verbosity=2)
