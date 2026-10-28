export interface Election {
  id: string;
  title: string;
  description: string;
  category: string;
  organizing_unit: string;
  status: string;
  starts_at: string;
  ends_at: string;
  result_policy: string;
  allowed_domains: string;
  require_roster: boolean;
  eligibility_frozen_at: string | null;
  eligibility_frozen_count: number;
  created_by_id: string;
  version: number;
  created_at: string;
  updated_at: string;
  positions: unknown[];
}
