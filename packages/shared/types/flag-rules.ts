export interface FlagRules {
  rollout_percentage: number;
  allowlist_user_ids: string[];
  blocklist_user_ids: string[];
  enabled_from: string | null;
  enabled_until: string | null;
  variants: string[];
  attributes: unknown;
}
