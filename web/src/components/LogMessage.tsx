import { useMemo } from 'react'

interface LogMessageProps {
  message: string
  level: string
  highlight?: string
}

const levelBg: Record<string, string> = {
  FATAL: '#fff1f0',
  ERROR: '#fff1f0',
  WARN: '#fffbe6',
  WARNING: '#fffbe6',
  INFO: '#f0f5ff',
  DEBUG: '#f6f6f6',
}

const levelColor: Record<string, string> = {
  FATAL: '#cf1322',
  ERROR: '#cf1322',
  WARN: '#d48806',
  WARNING: '#d48806',
  INFO: '#1677ff',
  DEBUG: '#8c8c8c',
}

function tryFormatJson(text: string): string | null {
  try {
    const obj = JSON.parse(text)
    return JSON.stringify(obj, null, 2)
  } catch {
    return null
  }
}

function escapeRegex(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export default function LogMessage({ message, level, highlight }: LogMessageProps) {
  const content = useMemo(() => {
    if (!message) return <span style={{ color: '#999' }}>-</span>

    // Try to detect embedded JSON
    const jsonMatch = message.match(/\{[\s\S]*\}|\[[\s\S]*\]/)
    let parts: React.ReactNode[] = []

    if (jsonMatch && jsonMatch.index !== undefined) {
      const before = message.substring(0, jsonMatch.index)
      const jsonStr = tryFormatJson(jsonMatch[0])
      const after = message.substring(jsonMatch.index + jsonMatch[0].length)

      if (jsonStr) {
        if (before) parts.push(renderHighlighted(before, level, highlight))
        parts.push(
          <pre
            key="json"
            style={{
              display: 'inline-block',
              verticalAlign: 'top',
              margin: '4px 0',
              padding: '6px 10px',
              background: '#fafafa',
              borderRadius: 4,
              fontSize: 11,
              fontFamily: 'Menlo, Monaco, Consolas, monospace',
              lineHeight: 1.5,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              border: '1px solid #f0f0f0',
              maxWidth: '100%',
              overflow: 'auto',
            }}
          >
            <span style={{ color: '#1677ff' }}>{jsonStr}</span>
          </pre>
        )
        if (after) parts.push(renderHighlighted(after, level, highlight))
        return <>{parts}</>
      }
    }

    return renderHighlighted(message, level, highlight)
  }, [message, level, highlight])

  return (
    <span
      style={{
        fontFamily: 'Menlo, Monaco, Consolas, monospace',
        fontSize: 12,
        lineHeight: 1.6,
        background: levelBg[level] || '#f6f6f6',
        padding: '2px 6px',
        borderRadius: 3,
        display: 'inline-block',
        maxWidth: '100%',
        color: levelColor[level] || '#333',
        borderLeft: `3px solid ${levelColor[level] || '#d9d9d9'}`,
      }}
    >
      {content}
    </span>
  )
}

function renderHighlighted(text: string, _level: string, highlight?: string): React.ReactNode {
  if (!highlight) return text

  const regex = new RegExp(`(${escapeRegex(highlight)})`, 'gi')
  const parts = text.split(regex)

  return parts.map((part, i) =>
    regex.test(part) ? (
      <mark key={i} style={{ background: '#ffeb3b', padding: '0 2px', borderRadius: 2 }}>
        {part}
      </mark>
    ) : (
      part
    )
  )
}
