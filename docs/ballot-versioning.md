# Ballot Versioning — ISSUE-06

## Overview

Ballot versioning ensures that every vote is interpreted against the exact ballot definition the voter saw when casting. This prevents silent changes to ballot definitions after voters have already made selections.

## Architecture

### BallotVersion Model

The `BallotVersion` table stores immutable snapshots of ballot definitions:

- **ID**: UUID primary key
- **ElectionID**: Reference to the election
- **Version**: Integer version number (1, 2, 3, ...)
- **ContentHash**: SHA-256 hash of the ballot content JSON
- **ContentJSON**: Serialized ballot structure (positions, candidates, selection rules)
- **PublishedBy**: User ID of the officer who published
- **PublishedAt**: Timestamp of publication
- **Status**: CURRENT or SUPERSEDED

### Content Structure

The `ContentJSON` field contains a `BallotContent` object with:

```typescript
{
  positions: [
    {
      id: string,
      title: string,
      description: string,
      question_type: "SINGLE_CHOICE" | "MULTI_CHOICE" | "YES_NO" | "REFERENDUM",
      min_selections: number,
      max_selections: number,
      seats: number,
      allow_blank: boolean,
      display_order: number,
      candidates: [
        {
          id: string,
          full_name: string,
          photo_url: string,
          manifesto: string,
          bio: string,
          ballot_number: number
        }
      ]
    }
  ]
}
```

## Publication Workflow

1. **Build**: Assemble positions and approved candidates into a ballot structure
2. **Validate**: Check all rules (minimum candidates, selection rules, no duplicates)
3. **Serialize**: Convert to JSON and compute SHA-256 hash
4. **Publish**: Create immutable BallotVersion record with CURRENT status
5. **Supersede**: Mark previous CURRENT versions as SUPERSEDED

## API Endpoints

### Ballot Version Management

- `POST /api/v1/staff/elections/:id/ballot/publish` - Publish new version
- `GET /api/v1/staff/elections/:id/ballot/versions` - List all versions
- `GET /api/v1/staff/elections/:id/ballot/versions/:version_id` - Get specific version
- `GET /api/v1/staff/elections/:id/ballot/versions/compare?v1=...&v2=...` - Compare two versions

### Position Management

- `GET /api/v1/staff/elections/:id/positions` - List positions
- `POST /api/v1/staff/elections/:id/positions` - Create position
- `PUT /api/v1/staff/elections/:id/positions/:position_id` - Update position (draft only)
- `DELETE /api/v1/staff/elections/:id/positions/:position_id` - Delete position (draft only)

### Candidate Management

- `GET /api/v1/staff/elections/:id/candidates` - List candidates (optional status filter)
- `POST /api/v1/staff/elections/:id/candidates` - Nominate candidate
- `PUT /api/v1/staff/elections/:id/candidates/:candidate_id` - Update candidate profile
- `POST /api/v1/staff/elections/:id/candidates/:candidate_id/approve` - Approve candidate
- `POST /api/v1/staff/elections/:id/candidates/:candidate_id/reject` - Reject candidate
- `POST /api/v1/staff/elections/:id/candidates/:candidate_id/withdraw` - Handle withdrawal

## Validation Rules

The ballot builder validates:

1. **Minimum Candidates**: Each position must have at least one approved candidate
2. **Selection Rules**: min_selections <= max_selections, seats >= 1
3. **Duplicate Ballot Numbers**: No two candidates in the same position can have the same ballot number
4. **Question Types**: Must be one of SINGLE_CHOICE, MULTI_CHOICE, YES_NO, REFERENDUM

## Candidate Workflow

1. **Nomination**: Created with status NOMINATED
2. **Review**: Officer reviews candidate profile
3. **Approval**: Status changed to APPROVED (now appears on ballot)
4. **Rejection**: Status changed to REJECTED (never appears on ballot)
5. **Withdrawal**: Status changed to WITHDRAWN (policy-dependent effect on published ballots)

## Privacy Considerations

- Ballot content stores only candidate public information (name, manifesto, photo)
- No voter data is stored in BallotVersion
- Votes reference BallotVersionID, not ElectionID directly, ensuring correct interpretation
- Hash provides cryptographic proof of content integrity

## Audit Logging

All ballot and candidate actions are logged to `ElectionAuditLog`:

- `BALLOT_PUBLISHED` - When a new ballot version is published
- `CANDIDATE_NOMINATED` - When a candidate is nominated
- `CANDIDATE_APPROVED` - When a candidate is approved
- `CANDIDATE_REJECTED` - When a candidate is rejected
- `CANDIDATE_WITHDRAWN` - When a candidate withdraws
- `POSITION_CREATED` - When a position is created
- `POSITION_UPDATED` - When a position is updated
- `POSITION_DELETED` - When a position is deleted

## UI Components

### Shared Components

- **BallotView**: Renders ballot content in preview or vote mode
  - Props: `content`, `mode` ("preview" | "vote"), `onSelectionChange`, `selections`
  - Preview mode: No selections stored, submit disabled
  - Vote mode: Stores selections in memory only, validates min/max

### Officer Views

- **Positions Manager** (`/staff/elections/:id/positions`) - CRUD for positions
- **Candidate Nominations Queue** (`/staff/elections/:id/candidates`) - Approve/reject candidates
- **Ballot Builder** (`/staff/elections/:id/ballot`) - Build and publish ballots with validation panel
- **Publish Dialog** - Shows version hash and warns about irreversibility
- **Ballot Versions** (`/staff/elections/:id/ballot/versions`) - List and compare versions
- **Ballot Preview** (`/staff/elections/:id/ballot/preview`) - Preview ballot as voter sees it

### Candidate Views

- **My Candidacy** (`/candidate`) - View own nominations, withdraw, see status

## Known Limitations

1. **Late Withdrawals**: If a candidate withdraws after voting starts, votes for that candidate are still counted. Policy decision needed for handling this case.
2. **File Upload**: Candidate photo upload validation is not yet implemented (pending ISSUE-06 task).
3. **Randomization**: Ballot order randomization is not yet implemented (can be added with per-voter random seed if needed).

## Testing

- Backend tests for validation logic
- Schema regression test for hash computation
- Browser tests for:
  - Officer builds ballot → validates → publishes → sees hash
  - Preview mode makes no vote/participation API calls
  - Version comparison shows correct diffs
  - Candidate nomination/approval workflow

## Future Enhancements

- Per-voter ballot randomization (if required by policy)
- Ballot version rollback capability (with audit trail)
- Automatic validation on position/candidate changes
- Real-time validation feedback in the builder UI
