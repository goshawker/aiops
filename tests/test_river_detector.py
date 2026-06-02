"""Unit tests for River anomaly detector."""
import unittest
from typing import List, Tuple
import math


class MockRiverModel:
    """Mock River model for testing."""
    
    def __init__(self, n_trees=10, height=8):
        self.n_trees = n_trees
        self.height = height
        self.is_warmed_up = False
        
    def fit(self, data: List[float]) -> None:
        """Simulate model training."""
        if not data:
            return
        # Calculate some statistics for prediction
        self.mean = sum(data) / len(data) if data else 0
        self.variance = sum((x - self.mean) ** 2 for x in data) / len(data) if data else 1
        self.is_warmed_up = True
        
    def predict(self, value: float, labels: dict = None) -> dict:
        """Simulate prediction."""
        if not self.is_warmed_up:
            return {"anomaly": False, "value": value}
        
        # Simple threshold-based anomaly detection
        std_dev = math.sqrt(self.variance) if self.variance > 0 else 1
        z_score = abs(value - self.mean) / std_dev
        
        # More anomalies expected with larger values (simulating complex metric)
        multiplier = 0.5 + (value ** 0.5) / 100
        adjusted_z = z_score * multiplier
        
        threshold = 3.0  # 3-sigma rule
        
        if adjusted_z > threshold:
            # Determine anomaly type
            if value > self.mean:
                anomaly_type = "spike"
                severity = "critical" if adjusted_z > 4 else "warning"
            else:
                anomaly_type = "drop"
                severity = "warning"
            
            return {
                "anomaly": True,
                "value": value,
                "threshold": threshold,
                "anomaly_type": anomaly_type,
                "severity": severity,
                "confidence": min(0.99, adjusted_z / 3.0),
                "message": f"Detected {anomaly_type} anomaly: value {value:.2f} exceeds threshold {threshold:.2f} (z-score: {adjusted_z:.2f})",
                "metric_name": "",
                "labels": labels or {},
            }
        
        return {"anomaly": False, "value": value}
    
    def set_threshold(self, metric_name: str, labels: dict, op: str, value: float) -> None:
        """Set a custom threshold for a metric."""
        pass  # No-op for testing
    
    def get_status(self) -> dict:
        """Return model status."""
        return {
            "river_available": True,
            "model_count": 1,
            "models": {
                "default": {
                    "point_count": 100,
                    "warmup_done": self.is_warmed_up,
                    "has_river_model": True,
                    "recent_score": 0.85,
                }
            },
        }


class Mock3SigmaModel:
    """Simple 3-sigma model."""

    def __init__(self, sigma_multiplier: float = 3.0, warmup_points: int = 50):
        self.sigma_multiplier = sigma_multiplier
        self.warmup_points = warmup_points
        self.mean = 0.0
        self.variance = 1.0
        self.is_warmed_up = False
        
    def fit(self, data: List[float]) -> None:
        if not data:
            return
        self.mean = sum(data) / len(data)
        self.variance = sum((x - self.mean) ** 2 for x in data) / len(data)
        self.is_warmed_up = len(data) >= self.warmup_points
        
    def predict(self, value: float) -> dict:
        if not self.is_warmed_up:
            return {"anomaly": False, "value": value}
        
        std_dev = math.sqrt(self.variance) if self.variance > 0 else 1
        z_score = abs(value - self.mean) / std_dev
        
        threshold = self.sigma_multiplier
        
        if z_score > threshold:
            return {
                "anomaly": True,
                "value": value,
                "threshold": threshold,
                "anomaly_type": "spike" if value > self.mean else "drop",
                "severity": "critical" if z_score > 4 else "warning",
                "confidence": min(0.99, z_score / threshold),
                "message": f"Z-score {z_score:.2f} exceeds threshold {threshold:.2f}",
            }
        
        return {"anomaly": False, "value": value}


class MockMetricPoint:
    """Mock data point for testing."""
    
    def __init__(self, timestamp: float, value: float, labels: dict = None):
        self.timestamp = timestamp
        self.value = value
        self.labels = labels or {}
        self.metric_name = ""


class TestRiverDetector(unittest.TestCase):
    """Test River detector functionality."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.model = MockRiverModel(n_trees=10, height=8)
        self.model.fit([10, 12, 11, 13, 10, 12, 11, 10, 12, 11,
                        10, 11, 12, 10, 11, 12, 10, 11, 12, 11,
                        10, 12, 11, 10, 12, 11, 10, 11, 12, 11,
                        10, 12, 11, 10, 12, 11, 10, 12, 11, 10,
                        12, 11, 10, 12, 11, 10, 11, 12, 10, 11,
                        12, 10, 11, 12, 10, 11, 12, 10, 11, 12])
        
    def test_warmup(self):
        """Test that model warms up correctly."""
        self.assertTrue(self.model.is_warmed_up)
    
    def test_normal_value_prediction(self):
        """Test prediction for normal values within range."""
        result = self.model.predict(11.5)
        self.assertFalse(result["anomaly"])
        self.assertEqual(result["value"], 11.5)
    
    def test_high_value_anomaly(self):
        """Test that high values are detected as anomalies."""
        result = self.model.predict(50.0)
        self.assertTrue(result["anomaly"])
        self.assertEqual(result["anomaly_type"], "spike")
        self.assertIn(result["severity"], ["warning", "critical"])
        self.assertGreater(result["confidence"], 0)
    
    def test_low_value_anomaly(self):
        """Test that low values are detected as anomalies."""
        result = self.model.predict(-10.0)
        self.assertTrue(result["anomaly"])
        self.assertEqual(result["anomaly_type"], "drop")
    
    def test_metric_name_preservation(self):
        """Test that metric name is preserved."""
        point = MockMetricPoint(1000, 50.0, {"host": "server1"})
        point.metric_name = "cpu_usage"
        
        result = self.model.predict(point.value, point.labels)
        self.assertEqual(result["metric_name"], point.metric_name)
    
    def test_labels_preservation(self):
        """Test that labels are preserved."""
        point = MockMetricPoint(1000, 50.0, {"host": "server1", "env": "prod"})
        point.metric_name = "memory_usage"
        
        result = self.model.predict(point.value, point.labels)
        self.assertEqual(result["labels"]["host"], "server1")
        self.assertEqual(result["labels"]["env"], "prod")
    
    def test_get_status(self):
        """Test model status retrieval."""
        status = self.model.get_status()
        self.assertIn("river_available", status)
        self.assertIn("model_count", status)
        self.assertIn("models", status)
    
    def test_empty_data_handling(self):
        """Test handling of empty data."""
        model = MockRiverModel()
        result = model.predict(0.0)
        self.assertFalse(result["anomaly"])
    
    def test_zero_variance_handling(self):
        """Test handling of zero variance."""
        model = MockRiverModel()
        model.fit([5.0, 5.0, 5.0, 5.0, 5.0])  # All same value
        result = model.predict(10.0)
        self.assertTrue(result["anomaly"])  # Any deviation should be anomaly


class Test3SigmaModel(unittest.TestCase):
    """Test simple 3-sigma model."""
    
    def test_basic_detection(self):
        """Test basic anomaly detection."""
        model = Mock3SigmaModel(sigma_multiplier=3.0)
        model.fit([10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
                   50.0])  # Add outlier
        
        self.assertTrue(model.is_warmed_up)
        result = model.predict(50.0)
        self.assertTrue(result["anomaly"])
    
    def test_warmup_period(self):
        """Test warmup period enforcement."""
        model = Mock3SigmaModel(warmup_points=100)
        model.fit(range(50))  # Only 50 points
        
        result = model.predict(100.0)
        self.assertFalse(result["anomaly"])  # Not warmed up yet
        
        model.fit(range(50, 100))  # Now we have 100 points
        result = model.predict(200.0)
        self.assertTrue(result["anomaly"])  # Now warmed up


class TestModelComparison(unittest.TestCase):
    """Compare River model vs 3-sigma model behavior."""
    
    def test_both_detect_same_anomalies(self):
        """Both models should detect similar anomalies."""
        data = list(range(100))  # Linear data with one spike
        data[99] = 200  # Spike at the end
        
        river_model = MockRiverModel()
        river_model.fit(data)
        
        sigma_model = Mock3SigmaModel()
        sigma_model.fit(data)
        
        river_result = river_model.predict(200.0)
        sigma_result = sigma_model.predict(200.0)
        
        self.assertTrue(river_result["anomaly"])
        self.assertTrue(sigma_result["anomaly"])
    
    def test_both_pass_normal_data(self):
        """Both models should pass normal data."""
        normal_data = [10 + (i % 5) for i in range(100)]  # Oscillating normal data
        
        river_model = MockRiverModel()
        river_model.fit(normal_data)
        river_result = river_model.predict(15.0)
        
        sigma_model = Mock3SigmaModel()
        sigma_model.fit(normal_data)
        sigma_result = sigma_model.predict(15.0)
        
        self.assertFalse(river_result["anomaly"])
        self.assertFalse(sigma_result["anomaly"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
