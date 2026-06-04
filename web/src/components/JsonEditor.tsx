// [vigilops] build:20260604
// VigilOps 天枢 - 智能运维平台
// Copyright (C) 2026 VigilOps Contributors
// Licensed under GPL-3.0 with Commercial Exception. See LICENSE for details.
import { useCallback, useRef } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'

interface JsonEditorProps {
  value?: string
  onChange?: (value: string) => void
  height?: number
  placeholder?: string
}

export default function JsonEditor({ value, onChange, height = 120, placeholder }: JsonEditorProps) {
  const heightRef = useRef(height)
  heightRef.current = height

  const handleMount: OnMount = useCallback((editor: editor.IStandaloneCodeEditor) => {
    const disposable = editor.onDidContentSizeChange(() => {
      const contentHeight = Math.min(Math.max(editor.getContentHeight(), heightRef.current), 400)
      const container = editor.getDomNode()
      if (container) {
        container.style.height = `${contentHeight}px`
        editor.layout()
      }
    })
    // Monaco auto-disposes when editor is destroyed, but explicit cleanup is cleaner
    editor.onDidDispose(() => disposable.dispose())
  }, [])

  return (
    <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, overflow: 'hidden', position: 'relative' }}>
      <Editor
        height={height}
        language="json"
        value={value ?? ''}
        onMount={handleMount}
        onChange={(v) => onChange?.(v ?? '')}
        theme="vs"
        options={{
          minimap: { enabled: false },
          lineNumbers: 'off',
          scrollBeyondLastLine: false,
          folding: true,
          wordWrap: 'on',
          tabSize: 2,
          fontSize: 13,
          renderLineHighlight: 'none',
          overviewRulerLanes: 0,
          scrollbar: { vertical: 'auto', horizontal: 'auto' },
          padding: { top: 8 },
        }}
      />
      {!value && placeholder && (
        <div style={{
          position: 'absolute', top: 8, left: 30, color: '#bfbfbf', fontSize: 13, pointerEvents: 'none',
        }}>
          {placeholder}
        </div>
      )}
    </div>
  )
}
