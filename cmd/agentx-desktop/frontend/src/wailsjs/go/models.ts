export namespace config {
	
	export class PeerMatch {
	    kind: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new PeerMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.id = source["id"];
	    }
	}
	export class BindingMatch {
	    channel: string;
	    account_id?: string;
	    peer?: PeerMatch;
	    guild_id?: string;
	    team_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new BindingMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel = source["channel"];
	        this.account_id = source["account_id"];
	        this.peer = this.convertValues(source["peer"], PeerMatch);
	        this.guild_id = source["guild_id"];
	        this.team_id = source["team_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentBinding {
	    agent_id: string;
	    match: BindingMatch;
	
	    static createFrom(source: any = {}) {
	        return new AgentBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent_id = source["agent_id"];
	        this.match = this.convertValues(source["match"], BindingMatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SubagentsConfig {
	    allow_agents?: string[];
	    model?: AgentModelConfig;
	
	    static createFrom(source: any = {}) {
	        return new SubagentsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allow_agents = source["allow_agents"];
	        this.model = this.convertValues(source["model"], AgentModelConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentModelConfig {
	    primary?: string;
	    fallbacks?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AgentModelConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.primary = source["primary"];
	        this.fallbacks = source["fallbacks"];
	    }
	}
	export class AgentConfig {
	    id: string;
	    default?: boolean;
	    name?: string;
	    workspace?: string;
	    model?: AgentModelConfig;
	    skills?: string[];
	    subagents?: SubagentsConfig;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.default = source["default"];
	        this.name = source["name"];
	        this.workspace = source["workspace"];
	        this.model = this.convertValues(source["model"], AgentModelConfig);
	        this.skills = source["skills"];
	        this.subagents = this.convertValues(source["subagents"], SubagentsConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AgentDefaults {
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
	
	    static createFrom(source: any = {}) {
	        return new AgentDefaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = source["workspace"];
	        this.restrict_to_workspace = source["restrict_to_workspace"];
	        this.provider = source["provider"];
	        this.model_name = source["model_name"];
	        this.model = source["model"];
	        this.model_fallbacks = source["model_fallbacks"];
	        this.image_model = source["image_model"];
	        this.image_model_fallbacks = source["image_model_fallbacks"];
	        this.max_tokens = source["max_tokens"];
	        this.temperature = source["temperature"];
	        this.max_tool_iterations = source["max_tool_iterations"];
	    }
	}
	
	export class AgentsConfig {
	    defaults: AgentDefaults;
	    list?: AgentConfig[];
	
	    static createFrom(source: any = {}) {
	        return new AgentsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaults = this.convertValues(source["defaults"], AgentDefaults);
	        this.list = this.convertValues(source["list"], AgentConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BraveConfig {
	    enabled: boolean;
	    api_key: string;
	    max_results: number;
	
	    static createFrom(source: any = {}) {
	        return new BraveConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.api_key = source["api_key"];
	        this.max_results = source["max_results"];
	    }
	}
	export class WeComAppConfig {
	    enabled: boolean;
	    corp_id: string;
	    corp_secret: string;
	    agent_id: number;
	    token: string;
	    encoding_aes_key: string;
	    webhook_host: string;
	    webhook_port: number;
	    webhook_path: string;
	    allow_from: string[];
	    reply_timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new WeComAppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.corp_id = source["corp_id"];
	        this.corp_secret = source["corp_secret"];
	        this.agent_id = source["agent_id"];
	        this.token = source["token"];
	        this.encoding_aes_key = source["encoding_aes_key"];
	        this.webhook_host = source["webhook_host"];
	        this.webhook_port = source["webhook_port"];
	        this.webhook_path = source["webhook_path"];
	        this.allow_from = source["allow_from"];
	        this.reply_timeout = source["reply_timeout"];
	    }
	}
	export class WeComConfig {
	    enabled: boolean;
	    token: string;
	    encoding_aes_key: string;
	    webhook_url: string;
	    webhook_host: string;
	    webhook_port: number;
	    webhook_path: string;
	    allow_from: string[];
	    reply_timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new WeComConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.token = source["token"];
	        this.encoding_aes_key = source["encoding_aes_key"];
	        this.webhook_url = source["webhook_url"];
	        this.webhook_host = source["webhook_host"];
	        this.webhook_port = source["webhook_port"];
	        this.webhook_path = source["webhook_path"];
	        this.allow_from = source["allow_from"];
	        this.reply_timeout = source["reply_timeout"];
	    }
	}
	export class OneBotConfig {
	    enabled: boolean;
	    ws_url: string;
	    access_token: string;
	    reconnect_interval: number;
	    group_trigger_prefix: string[];
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new OneBotConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.ws_url = source["ws_url"];
	        this.access_token = source["access_token"];
	        this.reconnect_interval = source["reconnect_interval"];
	        this.group_trigger_prefix = source["group_trigger_prefix"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class LINEConfig {
	    enabled: boolean;
	    channel_secret: string;
	    channel_access_token: string;
	    webhook_host: string;
	    webhook_port: number;
	    webhook_path: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new LINEConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.channel_secret = source["channel_secret"];
	        this.channel_access_token = source["channel_access_token"];
	        this.webhook_host = source["webhook_host"];
	        this.webhook_port = source["webhook_port"];
	        this.webhook_path = source["webhook_path"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class SlackConfig {
	    enabled: boolean;
	    bot_token: string;
	    app_token: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new SlackConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.bot_token = source["bot_token"];
	        this.app_token = source["app_token"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class DingTalkConfig {
	    enabled: boolean;
	    client_id: string;
	    client_secret: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new DingTalkConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.client_id = source["client_id"];
	        this.client_secret = source["client_secret"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class QQConfig {
	    enabled: boolean;
	    app_id: string;
	    app_secret: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new QQConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.app_id = source["app_id"];
	        this.app_secret = source["app_secret"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class MaixCamConfig {
	    enabled: boolean;
	    host: string;
	    port: number;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new MaixCamConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class DiscordConfig {
	    enabled: boolean;
	    token: string;
	    allow_from: string[];
	    mention_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiscordConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.token = source["token"];
	        this.allow_from = source["allow_from"];
	        this.mention_only = source["mention_only"];
	    }
	}
	export class FeishuConfig {
	    enabled: boolean;
	    app_id: string;
	    app_secret: string;
	    encrypt_key: string;
	    verification_token: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new FeishuConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.app_id = source["app_id"];
	        this.app_secret = source["app_secret"];
	        this.encrypt_key = source["encrypt_key"];
	        this.verification_token = source["verification_token"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class TelegramConfig {
	    enabled: boolean;
	    token: string;
	    proxy: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new TelegramConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.token = source["token"];
	        this.proxy = source["proxy"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class WhatsAppConfig {
	    enabled: boolean;
	    bridge_url: string;
	    allow_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new WhatsAppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.bridge_url = source["bridge_url"];
	        this.allow_from = source["allow_from"];
	    }
	}
	export class ChannelsConfig {
	    whatsapp: WhatsAppConfig;
	    telegram: TelegramConfig;
	    feishu: FeishuConfig;
	    discord: DiscordConfig;
	    maixcam: MaixCamConfig;
	    qq: QQConfig;
	    dingtalk: DingTalkConfig;
	    slack: SlackConfig;
	    line: LINEConfig;
	    onebot: OneBotConfig;
	    wecom: WeComConfig;
	    wecom_app: WeComAppConfig;
	
	    static createFrom(source: any = {}) {
	        return new ChannelsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.whatsapp = this.convertValues(source["whatsapp"], WhatsAppConfig);
	        this.telegram = this.convertValues(source["telegram"], TelegramConfig);
	        this.feishu = this.convertValues(source["feishu"], FeishuConfig);
	        this.discord = this.convertValues(source["discord"], DiscordConfig);
	        this.maixcam = this.convertValues(source["maixcam"], MaixCamConfig);
	        this.qq = this.convertValues(source["qq"], QQConfig);
	        this.dingtalk = this.convertValues(source["dingtalk"], DingTalkConfig);
	        this.slack = this.convertValues(source["slack"], SlackConfig);
	        this.line = this.convertValues(source["line"], LINEConfig);
	        this.onebot = this.convertValues(source["onebot"], OneBotConfig);
	        this.wecom = this.convertValues(source["wecom"], WeComConfig);
	        this.wecom_app = this.convertValues(source["wecom_app"], WeComAppConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ClawHubRegistryConfig {
	    enabled: boolean;
	    base_url: string;
	    auth_token: string;
	    search_path: string;
	    skills_path: string;
	    download_path: string;
	    timeout: number;
	    max_zip_size: number;
	    max_response_size: number;
	
	    static createFrom(source: any = {}) {
	        return new ClawHubRegistryConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.base_url = source["base_url"];
	        this.auth_token = source["auth_token"];
	        this.search_path = source["search_path"];
	        this.skills_path = source["skills_path"];
	        this.download_path = source["download_path"];
	        this.timeout = source["timeout"];
	        this.max_zip_size = source["max_zip_size"];
	        this.max_response_size = source["max_response_size"];
	    }
	}
	export class DevicesConfig {
	    enabled: boolean;
	    monitor_usb: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DevicesConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.monitor_usb = source["monitor_usb"];
	    }
	}
	export class HeartbeatConfig {
	    enabled: boolean;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new HeartbeatConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.interval = source["interval"];
	    }
	}
	export class SearchCacheConfig {
	    max_size: number;
	    ttl_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchCacheConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.max_size = source["max_size"];
	        this.ttl_seconds = source["ttl_seconds"];
	    }
	}
	export class SkillsRegistriesConfig {
	    clawhub: ClawHubRegistryConfig;
	
	    static createFrom(source: any = {}) {
	        return new SkillsRegistriesConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clawhub = this.convertValues(source["clawhub"], ClawHubRegistryConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SkillsToolsConfig {
	    registries: SkillsRegistriesConfig;
	    max_concurrent_searches: number;
	    search_cache: SearchCacheConfig;
	
	    static createFrom(source: any = {}) {
	        return new SkillsToolsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.registries = this.convertValues(source["registries"], SkillsRegistriesConfig);
	        this.max_concurrent_searches = source["max_concurrent_searches"];
	        this.search_cache = this.convertValues(source["search_cache"], SearchCacheConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExecConfig {
	    enable_deny_patterns: boolean;
	    custom_deny_patterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExecConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enable_deny_patterns = source["enable_deny_patterns"];
	        this.custom_deny_patterns = source["custom_deny_patterns"];
	    }
	}
	export class CronToolsConfig {
	    exec_timeout_minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new CronToolsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exec_timeout_minutes = source["exec_timeout_minutes"];
	    }
	}
	export class PerplexityConfig {
	    enabled: boolean;
	    api_key: string;
	    max_results: number;
	
	    static createFrom(source: any = {}) {
	        return new PerplexityConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.api_key = source["api_key"];
	        this.max_results = source["max_results"];
	    }
	}
	export class DuckDuckGoConfig {
	    enabled: boolean;
	    max_results: number;
	
	    static createFrom(source: any = {}) {
	        return new DuckDuckGoConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.max_results = source["max_results"];
	    }
	}
	export class TavilyConfig {
	    enabled: boolean;
	    api_key: string;
	    base_url: string;
	    max_results: number;
	
	    static createFrom(source: any = {}) {
	        return new TavilyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.max_results = source["max_results"];
	    }
	}
	export class WebToolsConfig {
	    brave: BraveConfig;
	    tavily: TavilyConfig;
	    duckduckgo: DuckDuckGoConfig;
	    perplexity: PerplexityConfig;
	    proxy?: string;
	
	    static createFrom(source: any = {}) {
	        return new WebToolsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brave = this.convertValues(source["brave"], BraveConfig);
	        this.tavily = this.convertValues(source["tavily"], TavilyConfig);
	        this.duckduckgo = this.convertValues(source["duckduckgo"], DuckDuckGoConfig);
	        this.perplexity = this.convertValues(source["perplexity"], PerplexityConfig);
	        this.proxy = source["proxy"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolsConfig {
	    web: WebToolsConfig;
	    cron: CronToolsConfig;
	    exec: ExecConfig;
	    skills: SkillsToolsConfig;
	
	    static createFrom(source: any = {}) {
	        return new ToolsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.web = this.convertValues(source["web"], WebToolsConfig);
	        this.cron = this.convertValues(source["cron"], CronToolsConfig);
	        this.exec = this.convertValues(source["exec"], ExecConfig);
	        this.skills = this.convertValues(source["skills"], SkillsToolsConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GatewayConfig {
	    host: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new GatewayConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	    }
	}
	export class ModelConfig {
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
	
	    static createFrom(source: any = {}) {
	        return new ModelConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	        this.model = source["model"];
	        this.api_base = source["api_base"];
	        this.api_key = source["api_key"];
	        this.proxy = source["proxy"];
	        this.auth_method = source["auth_method"];
	        this.connect_mode = source["connect_mode"];
	        this.workspace = source["workspace"];
	        this.rpm = source["rpm"];
	        this.max_tokens_field = source["max_tokens_field"];
	        this.request_timeout = source["request_timeout"];
	    }
	}
	export class OpenAIProviderConfig {
	    api_key: string;
	    api_base: string;
	    proxy?: string;
	    request_timeout?: number;
	    auth_method?: string;
	    connect_mode?: string;
	    web_search: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OpenAIProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.api_base = source["api_base"];
	        this.proxy = source["proxy"];
	        this.request_timeout = source["request_timeout"];
	        this.auth_method = source["auth_method"];
	        this.connect_mode = source["connect_mode"];
	        this.web_search = source["web_search"];
	    }
	}
	export class ProviderConfig {
	    api_key: string;
	    api_base: string;
	    proxy?: string;
	    request_timeout?: number;
	    auth_method?: string;
	    connect_mode?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.api_base = source["api_base"];
	        this.proxy = source["proxy"];
	        this.request_timeout = source["request_timeout"];
	        this.auth_method = source["auth_method"];
	        this.connect_mode = source["connect_mode"];
	    }
	}
	export class ProvidersConfig {
	    anthropic: ProviderConfig;
	    openai: OpenAIProviderConfig;
	    openrouter: ProviderConfig;
	    groq: ProviderConfig;
	    zhipu: ProviderConfig;
	    vllm: ProviderConfig;
	    gemini: ProviderConfig;
	    nvidia: ProviderConfig;
	    ollama: ProviderConfig;
	    moonshot: ProviderConfig;
	    shengsuanyun: ProviderConfig;
	    deepseek: ProviderConfig;
	    cerebras: ProviderConfig;
	    volcengine: ProviderConfig;
	    github_copilot: ProviderConfig;
	    antigravity: ProviderConfig;
	    qwen: ProviderConfig;
	    mistral: ProviderConfig;
	
	    static createFrom(source: any = {}) {
	        return new ProvidersConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anthropic = this.convertValues(source["anthropic"], ProviderConfig);
	        this.openai = this.convertValues(source["openai"], OpenAIProviderConfig);
	        this.openrouter = this.convertValues(source["openrouter"], ProviderConfig);
	        this.groq = this.convertValues(source["groq"], ProviderConfig);
	        this.zhipu = this.convertValues(source["zhipu"], ProviderConfig);
	        this.vllm = this.convertValues(source["vllm"], ProviderConfig);
	        this.gemini = this.convertValues(source["gemini"], ProviderConfig);
	        this.nvidia = this.convertValues(source["nvidia"], ProviderConfig);
	        this.ollama = this.convertValues(source["ollama"], ProviderConfig);
	        this.moonshot = this.convertValues(source["moonshot"], ProviderConfig);
	        this.shengsuanyun = this.convertValues(source["shengsuanyun"], ProviderConfig);
	        this.deepseek = this.convertValues(source["deepseek"], ProviderConfig);
	        this.cerebras = this.convertValues(source["cerebras"], ProviderConfig);
	        this.volcengine = this.convertValues(source["volcengine"], ProviderConfig);
	        this.github_copilot = this.convertValues(source["github_copilot"], ProviderConfig);
	        this.antigravity = this.convertValues(source["antigravity"], ProviderConfig);
	        this.qwen = this.convertValues(source["qwen"], ProviderConfig);
	        this.mistral = this.convertValues(source["mistral"], ProviderConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionConfig {
	    dm_scope?: string;
	    identity_links?: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new SessionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dm_scope = source["dm_scope"];
	        this.identity_links = source["identity_links"];
	    }
	}
	export class Config {
	    agents: AgentsConfig;
	    bindings?: AgentBinding[];
	    session?: SessionConfig;
	    channels: ChannelsConfig;
	    providers?: ProvidersConfig;
	    model_list: ModelConfig[];
	    gateway: GatewayConfig;
	    tools: ToolsConfig;
	    heartbeat: HeartbeatConfig;
	    devices: DevicesConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agents = this.convertValues(source["agents"], AgentsConfig);
	        this.bindings = this.convertValues(source["bindings"], AgentBinding);
	        this.session = this.convertValues(source["session"], SessionConfig);
	        this.channels = this.convertValues(source["channels"], ChannelsConfig);
	        this.providers = this.convertValues(source["providers"], ProvidersConfig);
	        this.model_list = this.convertValues(source["model_list"], ModelConfig);
	        this.gateway = this.convertValues(source["gateway"], GatewayConfig);
	        this.tools = this.convertValues(source["tools"], ToolsConfig);
	        this.heartbeat = this.convertValues(source["heartbeat"], HeartbeatConfig);
	        this.devices = this.convertValues(source["devices"], DevicesConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	

}

export namespace health {
	
	export class Check {
	    name: string;
	    status: string;
	    message?: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new Check(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatusResponse {
	    status: string;
	    uptime: string;
	    checks?: Record<string, Check>;
	
	    static createFrom(source: any = {}) {
	        return new StatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.uptime = source["uptime"];
	        this.checks = this.convertValues(source["checks"], Check, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class AppInfo {
	    version: string;
	    os: string;
	    arch: string;
	    configPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.configPath = source["configPath"];
	    }
	}
	export class BootstrapFile {
	    name: string;
	    path: string;
	    content: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.content = source["content"];
	        this.exists = source["exists"];
	    }
	}
	export class ChannelInfo {
	    name: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChannelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ChatResponse {
	    response: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.response = source["response"];
	        this.error = source["error"];
	    }
	}
	export class ModelInfo {
	    modelName: string;
	    model: string;
	    hasKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelName = source["modelName"];
	        this.model = source["model"];
	        this.hasKey = source["hasKey"];
	    }
	}
	export class GatewayStatus {
	    running: boolean;
	    health?: health.StatusResponse;
	    channels: ChannelInfo[];
	    models: ModelInfo[];
	
	    static createFrom(source: any = {}) {
	        return new GatewayStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.health = this.convertValues(source["health"], health.StatusResponse);
	        this.channels = this.convertValues(source["channels"], ChannelInfo);
	        this.models = this.convertValues(source["models"], ModelInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PlatformInfo {
	    os: string;
	    arch: string;
	    installDir: string;
	    binaryPath: string;
	    binaryExists: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.installDir = source["installDir"];
	        this.binaryPath = source["binaryPath"];
	        this.binaryExists = source["binaryExists"];
	        this.version = source["version"];
	    }
	}
	export class ProviderOption {
	    name: string;
	    id: string;
	    modelName: string;
	    model: string;
	    apiBase: string;
	    keyURL: string;
	    needsKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.id = source["id"];
	        this.modelName = source["modelName"];
	        this.model = source["model"];
	        this.apiBase = source["apiBase"];
	        this.keyURL = source["keyURL"];
	        this.needsKey = source["needsKey"];
	    }
	}
	export class SetupState {
	    binaryInstalled: boolean;
	    configExists: boolean;
	    hasApiKey: boolean;
	    hasChannel: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.binaryInstalled = source["binaryInstalled"];
	        this.configExists = source["configExists"];
	        this.hasApiKey = source["hasApiKey"];
	        this.hasChannel = source["hasChannel"];
	    }
	}
	export class SkillEntry {
	    name: string;
	    source: string;
	    description: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.description = source["description"];
	        this.path = source["path"];
	    }
	}

}

