export interface EligibilityImportBatch {
  id: string;
  election_id: string;
  imported_by_id: string;
  source: string;
  file_name: string;
  content_hash: string;
  total: number;
  created: number;
  updated: number;
  revoked: number;
  created_at: string;
}
