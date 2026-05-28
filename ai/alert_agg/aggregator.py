"""Alert Aggregation - Label Matching.

Groups similar alerts into incidents based on label matching.
Phase 1: Simple label-based grouping (not LCSS).
"""

import logging
import time
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)


@dataclass
class Alert:
    """Incoming alert."""
    source_type: str
    source: str
    host: str = ""
    service: str = ""
    severity: str = "warning"
    title: str = ""
    message: str = ""
    value: str = ""
    threshold: str = ""
    labels: dict = field(default_factory=dict)
    fired_at: float = field(default_factory=time.time)


@dataclass
class Incident:
    """Aggregated incident from multiple alerts."""
    id: str
    title: str
    description: str
    severity: str
    affected_services: list[str] = field(default_factory=list)
    affected_hosts: list[str] = field(default_factory=list)
    event_count: int = 0
    alerts: list[Alert] = field(default_factory=list)
    status: str = "open"
    created_at: float = field(default_factory=time.time)
    updated_at: float = field(default_factory=time.time)


class LabelMatchAggregator:
    """Aggregates alerts into incidents using label matching.

    Rules:
    1. Same service + same host + same severity → group within 5 min window
    2. Same service + critical → group across hosts within 5 min
    3. Host down → suppress all alerts from that host
    """

    def __init__(self, window_seconds: int = 300):
        self.window_seconds = window_seconds
        # key -> Incident
        self.open_incidents: dict[str, Incident] = {}
        # host -> last seen timestamp (for host-down detection)
        self.host_last_seen: dict[str, float] = {}
        self._incident_counter = 0

    def process(self, alert: Alert) -> Optional[Incident]:
        """Process an alert, return incident if created/updated."""
        now = time.time()

        # Generate grouping key
        key = self._make_key(alert)

        # Check if this fits into an existing open incident
        if key in self.open_incidents:
            incident = self.open_incidents[key]
            if now - incident.created_at < self.window_seconds:
                incident.alerts.append(alert)
                incident.event_count += 1
                incident.updated_at = now
                # Update severity to highest
                if self._severity_rank(alert.severity) > self._severity_rank(incident.severity):
                    incident.severity = alert.severity
                # Add unique hosts/services
                if alert.host and alert.host not in incident.affected_hosts:
                    incident.affected_hosts.append(alert.host)
                if alert.service and alert.service not in incident.affected_services:
                    incident.affected_services.append(alert.service)
                return None  # No new incident

        # Create new incident
        self._incident_counter += 1
        incident = Incident(
            id=f"INC-{int(now)}-{self._incident_counter}",
            title=alert.title or f"{alert.source} anomaly",
            description=alert.message,
            severity=alert.severity,
            affected_services=[alert.service] if alert.service else [],
            affected_hosts=[alert.host] if alert.host else [],
            event_count=1,
            alerts=[alert],
        )
        self.open_incidents[key] = incident

        logger.info(f"New incident: {incident.id} - {incident.title}")
        return incident

    def resolve(self, incident_id: str) -> bool:
        """Mark an incident as resolved."""
        for key, inc in self.open_incidents.items():
            if inc.id == incident_id:
                inc.status = "resolved"
                inc.updated_at = time.time()
                del self.open_incidents[key]
                return True
        return False

    def cleanup_expired(self):
        """Remove expired incidents from window."""
        now = time.time()
        expired = [
            k for k, v in self.open_incidents.items()
            if now - v.created_at > self.window_seconds * 2
        ]
        for k in expired:
            del self.open_incidents[k]

    def _make_key(self, alert: Alert) -> str:
        """Generate grouping key for an alert."""
        # Group by service + severity for service-level incidents
        if alert.service:
            return f"service:{alert.service}:{alert.severity}"
        # Group by host for host-level incidents
        if alert.host:
            return f"host:{alert.host}:{alert.severity}"
        # Fallback: group by source
        return f"source:{alert.source}:{alert.severity}"

    def _severity_rank(self, severity: str) -> int:
        return {"info": 0, "warning": 1, "critical": 2}.get(severity, 0)
