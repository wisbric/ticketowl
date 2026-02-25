// ── Ticket ──────────────────────────────────────────────────────────

export interface EnrichedTicket {
  id: number;
  number: string;
  title: string;
  state: string;
  state_id: number;
  priority: string;
  priority_id: number;
  group: string;
  group_id: number;
  owner: string;
  owner_id: number;
  customer_id: number;
  organization_id: number;
  tags: string[];
  created_at: string;
  updated_at: string;
  close_at?: string;
  meta_id?: string;
  sla_policy_id?: string;
}

export interface CreateTicketRequest {
  title: string;
  group_id: number;
  customer_id?: number;
  state_id?: number;
  priority_id?: number;
  body?: string;
}

export interface UpdateTicketRequest {
  state_id?: number;
  priority_id?: number;
  owner_id?: number;
  group_id?: number;
}

// ── Comments / Thread ───────────────────────────────────────────────

export interface ThreadEntry {
  id: string;
  source: string;
  type: string;
  sender: string;
  body: string;
  internal: boolean;
  created_by: string;
  created_at: string;
}

export interface AddReplyRequest {
  body: string;
  content_type?: string;
}

export interface AddNoteRequest {
  body: string;
}

// ── SLA ─────────────────────────────────────────────────────────────

export type SLAStateLabel = "on_track" | "warning" | "breached" | "met";

export interface SLAPolicy {
  id: string;
  name: string;
  priority: string;
  response_minutes: number;
  resolution_minutes: number;
  warning_threshold: number;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface SLAState {
  id: string;
  ticket_meta_id: string;
  response_due_at?: string;
  resolution_due_at?: string;
  response_met_at?: string;
  first_breach_alerted_at?: string;
  state: SLAStateLabel;
  paused: boolean;
  paused_at?: string;
  accumulated_pause_secs: number;
  updated_at: string;
}

export interface CreateSLAPolicyRequest {
  name: string;
  priority: string;
  response_minutes: number;
  resolution_minutes: number;
  warning_threshold?: number;
  is_default?: boolean;
}

export interface UpdateSLAPolicyRequest {
  name?: string;
  response_minutes?: number;
  resolution_minutes?: number;
  warning_threshold?: number;
  is_default?: boolean;
}

// ── Links ───────────────────────────────────────────────────────────

export interface TicketLinks {
  incidents: IncidentLink[];
  articles: ArticleLink[];
  postmortem?: PostMortemLink;
}

export interface IncidentLink {
  id: string;
  ticket_meta_id: string;
  incident_id: string;
  incident_slug: string;
  linked_by: string;
  created_at: string;
}

export interface ArticleLink {
  id: string;
  ticket_meta_id: string;
  article_id: string;
  article_slug: string;
  article_title: string;
  linked_by: string;
  created_at: string;
}

export interface PostMortemLink {
  id: string;
  ticket_meta_id: string;
  postmortem_id: string;
  postmortem_url: string;
  created_by: string;
  created_at: string;
}

export interface CreateIncidentLinkRequest {
  incident_id: string;
}

export interface CreateArticleLinkRequest {
  article_id: string;
}

export interface PostMortemResult {
  postmortem_id: string;
  postmortem_url: string;
}

// ── Suggestions ─────────────────────────────────────────────────────

export interface Suggestion {
  id: string;
  slug: string;
  title: string;
  excerpt: string;
  tags: string[];
  url: string;
}

// ── Customer Portal ─────────────────────────────────────────────────

export interface PortalTicket {
  id: number;
  number: string;
  title: string;
  status: string;
  priority: string;
  sla_state?: string;
  created_at: string;
  updated_at: string;
}

export interface PortalTicketDetail extends PortalTicket {
  comments: PortalComment[];
  linked_articles: PortalArticle[];
  response_due_at?: string;
  resolution_due_at?: string;
}

export interface PortalComment {
  id: number;
  body: string;
  sender: string;
  created_by: string;
  created_at: string;
}

export interface PortalArticle {
  id: string;
  slug: string;
  title: string;
}

export interface PortalReplyRequest {
  body: string;
}

// ── Admin ───────────────────────────────────────────────────────────

export interface ConfigOverview {
  zammad?: ZammadConfigSummary;
  nightowl?: IntegrationKeySummary;
  bookowl?: IntegrationKeySummary;
}

export interface ZammadConfigSummary {
  url: string;
  pause_statuses: string[];
  updated_at: string;
}

export interface UpdateZammadConfigRequest {
  url: string;
  api_token: string;
  webhook_secret?: string;
  pause_statuses?: string[];
}

export interface TestZammadRequest {
  url: string;
  api_token: string;
}

export interface TestZammadResult {
  success: boolean;
  error?: string;
}

export interface IntegrationKeySummary {
  api_url: string;
  updated_at: string;
}

export interface UpdateIntegrationKeyRequest {
  api_key: string;
  api_url: string;
}

export interface CustomerOrg {
  id: string;
  name: string;
  oidc_group: string;
  zammad_org_id?: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerOrgRequest {
  name: string;
  oidc_group: string;
  zammad_org_id?: number;
}

export interface UpdateCustomerOrgRequest {
  name?: string;
  oidc_group?: string;
  zammad_org_id?: number;
}

export interface AutoTicketRule {
  id: string;
  name: string;
  enabled: boolean;
  alert_group?: string;
  min_severity?: string;
  default_priority: string;
  default_group?: string;
  title_template: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAutoTicketRuleRequest {
  name: string;
  enabled?: boolean;
  alert_group?: string;
  min_severity?: string;
  default_priority: string;
  default_group?: string;
  title_template: string;
}

export interface UpdateAutoTicketRuleRequest {
  name?: string;
  enabled?: boolean;
  alert_group?: string;
  min_severity?: string;
  default_priority?: string;
  default_group?: string;
  title_template?: string;
}
