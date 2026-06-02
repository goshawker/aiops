import { create } from 'zustand'

export type ModelProvider = 'ollama' | 'openai-compatible' | 'local'

export interface ModelConfig {
  provider: ModelProvider
  // Ollama
  ollamaEndpoint: string
  ollamaModel: string
  // OpenAI-compatible
  apiEndpoint: string
  apiKey: string
  apiModel: string
  // Local
  localModelPath: string
  // Common
  temperature: number
  maxTokens: number
  contextLength: number
  timeout: number
  // Status
  connected: boolean
  lastTestTime: string | null
}

interface ModelConfigState {
  config: ModelConfig
  updateConfig: (partial: Partial<ModelConfig>) => void
  resetConfig: () => void
  getModelLabel: () => string
}

const STORAGE_KEY = 'aiops-model-config'

const defaultConfig: ModelConfig = {
  provider: 'ollama',
  ollamaEndpoint: 'http://localhost:11434',
  ollamaModel: 'qwen2:7b',
  apiEndpoint: '',
  apiKey: '',
  apiModel: 'gpt-4o-mini',
  localModelPath: '',
  temperature: 0.7,
  maxTokens: 2048,
  contextLength: 4096,
  timeout: 120,
  connected: false,
  lastTestTime: null,
}

const loadConfig = (): ModelConfig => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) return { ...defaultConfig, ...JSON.parse(saved) }
  } catch { /* ignore */ }
  return defaultConfig
}

export const useModelConfigStore = create<ModelConfigState>((set, get) => ({
  config: loadConfig(),

  updateConfig: (partial) => {
    const newConfig = { ...get().config, ...partial }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(newConfig))
    set({ config: newConfig })
  },

  resetConfig: () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaultConfig))
    set({ config: defaultConfig })
  },

  getModelLabel: () => {
    const c = get().config
    switch (c.provider) {
      case 'ollama': return `${c.ollamaModel} (Ollama)`
      case 'openai-compatible': return `${c.apiModel} (API)`
      case 'local': return `本地模型`
      default: return '未配置'
    }
  },
}))
