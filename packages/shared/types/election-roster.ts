export interface ElectionRoster {
  id: string;
  election_id: string;
  student_number: string;
  reg_number: string;
  email: string;
  full_name: string;
  faculty: string;
  department: string;
  programme: string;
  study_year: number;
  status: string;
  source: string;
  import_batch_id: string;
  imported_by_id: string;
  status_reason: string;
  status_changed_at: string | null;
  created_at: string;
  updated_at: string;
}
