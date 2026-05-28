"""Health Summary Engine - Rule Engine (Template-based).

Generates health summaries using templates and rules.
Phase 1: Rule engine (this file)
Phase 2: Qwen2-1.5B (will extend this interface)
"""

import logging
import time
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)


@dataclass
class MetricSnapshot:
    """Snapshot of a metric at a point in time."""
    name: str
    value: float
    unit: str = ""
    change_pct: float = 0.0  # % change over last hour
    host: str = ""
    service: str = ""


@dataclass
class LogSummary:
    """Summary of recent logs."""
    total_count: int = 0
    error_count: int = 0
    warn_count: int = 0
    top_errors: list[str] = field(default_factory=list)


@dataclass
class HealthReport:
    """Generated health report."""
    status: str  # "normal", "warning", "critical"
    summary: str  # One-line summary
    details: list[str] = field(default_factory=list)
    recommendations: list[str] = field(default_factory=list)
    source: str = "rule_engine"  # "rule_engine" or "qwen2"
    generated_at: float = field(default_factory=time.time)


class RuleEngineSummary:
    """Template-based health summary generator.

    Generates Chinese health summaries using rule-based templates.
    """

    TEMPLATES = {
        "normal": "系统整体正常。{details}",
        "warning": "系统存在警告：{details}",
        "critical": "检测到异常：{details}。建议立即检查。",
    }

    def generate(
        self,
        metrics: list[MetricSnapshot],
        log_summary: Optional[LogSummary] = None,
        incidents: list[dict] = None,
    ) -> HealthReport:
        """Generate a health report from current state."""
        details = []
        recommendations = []
        status = "normal"

        # Analyze metrics
        for m in metrics:
            if m.name == "cpu_usage" and m.value > 80:
                status = self._escalate(status, "warning")
                detail = f"CPU 使用率 {m.value:.0f}%"
                if m.change_pct > 10:
                    detail += f"（近 1 小时上升 {m.change_pct:.0f}%）"
                    recommendations.append("关注 CPU 使用趋势，可能存在资源泄漏")
                details.append(detail)

            elif m.name == "memory_usage" and m.value > 85:
                status = self._escalate(status, "warning")
                details.append(f"内存使用率 {m.value:.0f}%")
                if m.value > 95:
                    status = self._escalate(status, "critical")
                    recommendations.append("内存即将耗尽，建议检查内存泄漏或扩容")

            elif m.name == "disk_usage" and m.value > 90:
                status = self._escalate(status, "warning")
                details.append(f"磁盘使用率 {m.value:.0f}%")
                recommendations.append("磁盘空间不足，建议清理日志或扩容")

            elif m.name == "error_rate" and m.value > 5:
                status = self._escalate(status, "critical")
                details.append(f"错误率 {m.value:.1f}%")
                recommendations.append("错误率异常升高，建议检查应用日志")

        # Analyze logs
        if log_summary:
            if log_summary.error_count > 10:
                status = self._escalate(status, "warning")
                details.append(f"过去 10 分钟错误日志 {log_summary.error_count} 条")
                if log_summary.top_errors:
                    details.append(f"高频错误：{log_summary.top_errors[0]}")
            elif log_summary.error_count == 0:
                details.append("过去 10 分钟无错误日志")

        # Analyze incidents
        if incidents:
            critical = sum(1 for i in incidents if i.get("severity") == "critical")
            if critical > 0:
                status = self._escalate(status, "critical")
                details.append(f"当前 {critical} 个严重事件")
            elif len(incidents) > 0:
                details.append(f"当前 {len(incidents)} 个活跃事件")

        # Generate summary
        if not details:
            details.append("所有指标正常")

        template = self.TEMPLATES.get(status, self.TEMPLATES["normal"])
        summary = template.format(details="；".join(details))

        return HealthReport(
            status=status,
            summary=summary,
            details=details,
            recommendations=recommendations,
            source="rule_engine",
        )

    def _escalate(self, current: str, new: str) -> str:
        """Escalate status to higher level."""
        levels = {"normal": 0, "warning": 1, "critical": 2}
        if levels.get(new, 0) > levels.get(current, 0):
            return new
        return current
