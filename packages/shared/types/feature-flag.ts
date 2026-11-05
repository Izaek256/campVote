export interface FeatureFlag {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  rules: unknown;
  created_at: string;
  updated_at: string;
  version: number;
}
