import { useCallback } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'

interface JsonEditorProps {
  value?: string
  onChange?: (value: string) => void
  height?: number
  placeholder?: string
}

export default function JsonEditor({ value, onChange, height = 120, placeholder }: JsonEditorProps) {
  const handleMount: OnMount = useCallback((editor: editor.IStandaloneCodeEditor) => {
    // Make the editor adjust to container width
    editor.onDidContentSizeChange(() => {
      const contentHeight = Math.min(Math.max(editor.getContentHeight(), height), 400)
      const container = editor.getDomNode()
      if (container) {
        container.style.height = `${contentHeight}px`
        editor.layout()
      }
    })
  }, [height])

  return (
    <div style={{ border: '1px solid #d9d9d9', borderRadius: 6, overflow: 'hidden' }}>
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
