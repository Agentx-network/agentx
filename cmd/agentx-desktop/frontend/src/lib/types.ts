// Mirrors Go backend structs

export interface AppInfo {
  version: string;
  os: string;
  arch: string;
  configPath: string;
}

export interface PlatformInfo {
  os: string;
  arch: string;
  installDir: string;
  binaryPath: string;
  binaryExists: boolean;
  version: string;
}

export interface GatewayStatus {
  running: boolean;
  health?: StatusResponse;
  channels: ChannelInfo[];
  models: ModelInfo[];
}

export interface StatusResponse {
  status: string;
  uptime: string;
  checks?: Record<string, HealthCheck>;
}

export interface HealthCheck {
  name: string;
  status: string;
  message?: string;
  timestamp: string;
}

export interface ChannelInfo {
  name: string;
  enabled: boolean;
}

export interface ModelInfo {
  modelName: string;
  model: string;
  hasKey: boolean;
}

export interface ModelConfig {
  model_name: string;
  model: string;
  api_base?: string;
  api_key: string;
  proxy?: string;
  auth_method?: string;
  connect_mode?: string;
  workspace?: string;
  rpm?: number;
  max_tokens_field?: string;
  request_timeout?: number;
}

export interface AgentDefaults {
  workspace: string;
  restrict_to_workspace: boolean;
  provider: string;
  model_name?: string;
  model?: string;
  model_fallbacks?: string[];
  image_model?: string;
  image_model_fallbacks?: string[];
  max_tokens: number;
  temperature?: number;
  max_tool_iterations: number;
}

export interface ProviderOption {
  name: string;
  id: string;
  modelName: string;
  model: string;
  apiBase: string;
  keyURL: string;
  needsKey: boolean;
}

export interface DownloadProgress {
  downloaded: number;
  total: number;
  percent: number;
}

export interface SetupState {
  binaryInstalled: boolean;
  configExists: boolean;
  hasApiKey: boolean;
}

export type Page = "installer" | "onboard" | "dashboard" | "config";
