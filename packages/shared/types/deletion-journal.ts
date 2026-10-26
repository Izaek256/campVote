export interface DeletionJournal {
  id: string;
  deleted_user_id: string;
  actor_id: string;
  actor_email: string;
  reason: string;
  records_affected: number;
  counts: string;
  prev_hash: string;
  hash: string;
  created_at: string;
}
