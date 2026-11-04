"use client";

/**
 * Design option A, the "certified ledger", as a React document.
 *
 * This is the results report a returning officer prints and signs. It is
 * rendered in the browser from exactly what the results screen already
 * publishes: the stored snapshot. The word "Elected" is only ever used when
 * that snapshot is CERTIFIED; until then the leading candidate is "Leading".
 *
 * Content flows to as many pages as it needs — a position block is never split
 * across a page break — and nothing here identifies a voter.
 */
import { Document, Page, StyleSheet, Text, View } from "@react-pdf/renderer";
import { categoryLabel } from "../format";
import type {
  Election,
  FacultyTurnout,
  PositionResult,
  ResultRow,
  ResultSnapshot,
} from "../types";
import {
  clockOnly,
  dayStamp,
  INK,
  INK_2,
  INK_3,
  LINE,
  MARGIN_X,
  ON_INK,
  ON_INK_DIM,
  ON_INK_LINE,
  percent,
  PRIMARY,
  seatLabel,
  stamp,
  SUNKEN,
  thousands,
  votesWord,
  WHITE,
} from "./theme";

interface Props {
  election: Election;
  snapshot: ResultSnapshot;
  /** Participation per faculty. Empty when the server has none to report. */
  faculties?: FacultyTurnout[];
}

const styles = StyleSheet.create({
  page: {
    backgroundColor: WHITE,
    color: INK,
    fontFamily: "Helvetica",
    fontSize: 9,
    paddingBottom: 52,
  },

  // The ink header band, printed on the first page only.
  band: { backgroundColor: INK, paddingHorizontal: MARGIN_X, paddingVertical: 22 },
  bandRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  wordmark: { color: WHITE, fontSize: 12, fontFamily: "Helvetica-Bold", letterSpacing: 0.3 },
  badge: {
    borderWidth: 1,
    borderColor: ON_INK_LINE,
    paddingVertical: 3,
    paddingHorizontal: 8,
  },
  badgeText: { color: WHITE, fontSize: 6.5, letterSpacing: 1.6 },
  eyebrow: { color: ON_INK, fontSize: 7.5, letterSpacing: 1.6, marginTop: 20 },
  bandTitle: {
    color: WHITE,
    fontSize: 20,
    fontFamily: "Helvetica-Bold",
    marginTop: 8,
    lineHeight: 1.15,
  },
  bandMeta: { color: ON_INK_DIM, fontSize: 8, marginTop: 8, letterSpacing: 0.2 },

  body: { paddingHorizontal: MARGIN_X },

  // The four-cell summary strip.
  strip: {
    flexDirection: "row",
    borderTopWidth: 1,
    borderBottomWidth: 1,
    borderColor: LINE,
    marginTop: 22,
  },
  cell: { flex: 1, paddingVertical: 12, paddingHorizontal: 12 },
  cellDivider: { borderLeftWidth: 1, borderColor: LINE },
  cellLabel: { fontSize: 6.5, letterSpacing: 1.2, color: INK_3 },
  cellValue: { fontSize: 19, fontFamily: "Helvetica-Bold", color: INK, marginTop: 6 },
  cellNote: { fontSize: 6.5, color: INK_3, marginTop: 4 },

  sectionHead: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-end",
    marginTop: 26,
    marginBottom: 2,
  },
  sectionTitle: { fontSize: 13, fontFamily: "Helvetica-Bold", color: INK },
  sectionNote: { fontSize: 6.5, letterSpacing: 1, color: INK_3 },

  position: { borderTopWidth: 1, borderColor: LINE, paddingTop: 13, paddingBottom: 12 },
  positionHead: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-end",
    marginBottom: 8,
  },
  positionTitle: { fontSize: 12, fontFamily: "Helvetica-Bold", color: INK },
  positionMeta: { fontSize: 6.5, letterSpacing: 1, color: INK_3 },

  candidate: { marginBottom: 9 },
  candidateLine: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 4,
  },
  candidateLeft: { flexDirection: "row", alignItems: "center", flexShrink: 1 },
  candidateName: { fontSize: 9.5, color: INK },
  score: { fontSize: 9.5, color: INK, fontFamily: "Helvetica-Bold" },

  chip: {
    backgroundColor: PRIMARY,
    paddingVertical: 1.5,
    paddingHorizontal: 5,
    marginLeft: 6,
  },
  chipText: { fontSize: 6, letterSpacing: 1, color: WHITE },
  chipOutline: {
    borderWidth: 1,
    borderColor: INK_3,
    paddingVertical: 1,
    paddingHorizontal: 5,
    marginLeft: 6,
  },
  chipOutlineText: { fontSize: 6, letterSpacing: 1, color: INK_3 },

  track: { height: 3, backgroundColor: SUNKEN },
  fill: { height: 3 },

  footnotes: { flexDirection: "row", justifyContent: "space-between", marginTop: 8 },
  footnote: { fontSize: 6.5, color: INK_3 },

  // The participation table. It mirrors the notice's table grammar: a sunken
  // header, then one hairline-ruled row per group.
  tableHead: {
    flexDirection: "row",
    backgroundColor: SUNKEN,
    paddingVertical: 7,
    paddingHorizontal: 8,
    marginTop: 8,
  },
  tableHeadCell: { fontSize: 6.5, letterSpacing: 1, color: INK_3 },
  tableRow: {
    flexDirection: "row",
    paddingVertical: 7,
    paddingHorizontal: 8,
    borderBottomWidth: 1,
    borderColor: LINE,
    alignItems: "center",
  },
  tableCell: { fontSize: 8.5, color: INK },
  tableCellMuted: { fontSize: 7.5, color: INK_3, fontStyle: "italic" },
  tableTotal: { borderBottomWidth: 0, borderTopWidth: 1.5, borderColor: INK },
  tableTotalCell: { fontSize: 8.5, color: INK, fontFamily: "Helvetica-Bold" },
  colFaculty: { flex: 3 },
  colNumber: { flex: 1.6, textAlign: "right" },
  colSpan: { flex: 4.8, textAlign: "right", paddingLeft: 8 },

  hashBox: {
    marginTop: 22,
    borderWidth: 1,
    borderColor: LINE,
    backgroundColor: SUNKEN,
    paddingVertical: 8,
    paddingHorizontal: 10,
  },
  hashText: { fontSize: 6.5, color: INK_2, fontFamily: "Courier" },

  signatures: { flexDirection: "row", marginTop: 26 },
  signature: { flex: 1, paddingRight: 24 },
  signatureRole: { fontSize: 6.5, letterSpacing: 1.2, color: INK_3 },
  signatureName: { fontSize: 9.5, color: INK, marginTop: 22 },
  signatureRule: { borderTopWidth: 1, borderColor: INK, marginTop: 3 },
  signatureHint: { fontSize: 6.5, color: INK_3, marginTop: 4 },

  footer: {
    position: "absolute",
    left: MARGIN_X,
    right: MARGIN_X,
    bottom: 24,
    borderTopWidth: 1,
    borderColor: LINE,
    paddingTop: 7,
    flexDirection: "row",
    justifyContent: "space-between",
  },
  footerGroup: { flexDirection: "row" },
  footerText: { fontSize: 6.5, color: INK_3 },
  footerSep: { fontSize: 6.5, color: LINE, paddingHorizontal: 6 },
});

/** Only a certified snapshot can call a winner "Elected". */
function outcomeNote(position: PositionResult, certified: boolean): string | null {
  if (position.outcome === "TIE_UNRESOLVED") return "Tied — no winner declared";
  if (position.outcome === "QUORUM_NOT_MET") return "Quorum not met";
  if (position.outcome === "NO_VOTES") return "No votes cast for this position";
  if (!certified && position.winners.length > 0) return "Leading";
  return null;
}

function CandidateRow({ row, certified }: { row: ResultRow; certified: boolean }) {
  const share = Math.max(0, Math.min(100, row.share_percent));
  const label = row.is_winner ? (certified ? "Elected" : "Leading") : null;
  return (
    <View style={styles.candidate} wrap={false}>
      <View style={styles.candidateLine}>
        <View style={styles.candidateLeft}>
          <Text style={styles.candidateName}>{row.name}</Text>
          {label ? (
            <View style={styles.chip}>
              <Text style={styles.chipText}>{label.toUpperCase()}</Text>
            </View>
          ) : null}
          {row.is_tie && !row.is_winner ? (
            <View style={styles.chipOutline}>
              <Text style={styles.chipOutlineText}>TIED</Text>
            </View>
          ) : null}
        </View>
        <Text style={styles.score}>
          {thousands(row.votes)} • {percent(row.share_percent)}
        </Text>
      </View>
      <View style={styles.track}>
        <View
          style={[
            styles.fill,
            { width: `${share}%`, backgroundColor: row.is_winner ? PRIMARY : INK_3 },
          ]}
        />
      </View>
    </View>
  );
}

function PositionBlock({ position, certified }: { position: PositionResult; certified: boolean }) {
  const note = outcomeNote(position, certified);
  return (
    <View style={styles.position} wrap={false}>
      <View style={styles.positionHead}>
        <Text style={styles.positionTitle}>{position.title}</Text>
        <Text style={styles.positionMeta}>
          {seatLabel(position.seats)} • {position.valid_votes.toLocaleString("en-GB")} VALID VOTES
        </Text>
      </View>
      {position.rows.map((row) => (
        <CandidateRow key={row.candidate_id} row={row} certified={certified} />
      ))}
      <View style={styles.footnotes}>
        <Text style={styles.footnote}>Percent of {votesWord(position.valid_votes)}.</Text>
        <Text style={styles.footnote}>
          {thousands(position.blank_count)} ballots left this position blank
          {position.abstain_count > 0 ? ` • ${thousands(position.abstain_count)} abstained` : ""}.
        </Text>
      </View>
      {note ? <Text style={[styles.footnote, { marginTop: 4 }]}>{note}</Text> : null}
    </View>
  );
}

export function ResultsLedgerDocument({ election, snapshot, faculties = [] }: Props) {
  const certified = snapshot.status === "CERTIFIED";
  const positions = snapshot.positions ?? [];

  return (
    <Document
      title={`Results • ${election.title}`}
      author="campVote Electoral Commission"
      subject={`Election ${election.id}`}
      creator="campVote"
    >
      <Page size="A4" style={styles.page} wrap>
        <View style={styles.band}>
          <View style={styles.bandRow}>
            <Text style={styles.wordmark}>campVote</Text>
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{certified ? "CERTIFIED" : "PROVISIONAL"}</Text>
            </View>
          </View>
          <Text style={styles.eyebrow}>OFFICIAL ELECTION RESULTS • {election.id}</Text>
          <Text style={styles.bandTitle}>{election.title}</Text>
          <Text style={styles.bandMeta}>
            {categoryLabel(election.category)} • {election.organizing_unit} • {positions.length}{" "}
            {positions.length === 1 ? "position" : "positions"} • Generated{" "}
            {stamp(snapshot.computed_at)}
          </Text>
        </View>

        <View style={styles.body}>
          <View style={styles.strip}>
            <View style={styles.cell}>
              <Text style={styles.cellLabel}>ELIGIBLE VOTERS</Text>
              <Text style={styles.cellValue}>{thousands(snapshot.eligible_voters)}</Text>
              <Text style={styles.cellNote}>on the frozen roster</Text>
            </View>
            <View style={[styles.cell, styles.cellDivider]}>
              <Text style={styles.cellLabel}>BALLOTS ACCEPTED</Text>
              <Text style={styles.cellValue}>{thousands(snapshot.ballots_cast)}</Text>
              <Text style={styles.cellNote}>closed {clockOnly(election.ends_at)}</Text>
            </View>
            <View style={[styles.cell, styles.cellDivider]}>
              <Text style={styles.cellLabel}>TURNOUT</Text>
              <Text style={styles.cellValue}>{percent(snapshot.turnout_percent)}</Text>
              <Text style={styles.cellNote}>of {thousands(snapshot.eligible_voters)}</Text>
            </View>
            <View style={[styles.cell, styles.cellDivider]}>
              <Text style={styles.cellLabel}>POSITIONS</Text>
              <Text style={styles.cellValue}>{positions.length}</Text>
              <Text style={styles.cellNote}>
                {certified ? `${positions.length} certified` : "provisional count"}
              </Text>
            </View>
          </View>

          <View style={styles.sectionHead}>
            <Text style={styles.sectionTitle}>Results by position</Text>
            <Text style={styles.sectionNote}>VALID VOTES ONLY • BLANKS NOTED</Text>
          </View>

          {positions.map((position) => (
            <PositionBlock key={position.position_id} position={position} certified={certified} />
          ))}

          {faculties.length > 0 ? (
            <>
              <View style={styles.sectionHead} minPresenceAhead={70}>
                <Text style={styles.sectionTitle}>Turnout by faculty</Text>
                <Text style={styles.sectionNote}>PERCENT OF EACH FACULTY ROSTER</Text>
              </View>
              <View style={styles.tableHead} wrap={false}>
                <Text style={[styles.tableHeadCell, styles.colFaculty]}>FACULTY</Text>
                <Text style={[styles.tableHeadCell, styles.colNumber]}>ELIGIBLE</Text>
                <Text style={[styles.tableHeadCell, styles.colNumber]}>BALLOTS</Text>
                <Text style={[styles.tableHeadCell, styles.colNumber]}>TURNOUT</Text>
              </View>
              {faculties.map((row) => (
                <View key={row.faculty} style={styles.tableRow} wrap={false}>
                  <Text style={[styles.tableCell, styles.colFaculty]}>{row.faculty}</Text>
                  {row.suppressed ? (
                    <Text style={[styles.tableCellMuted, styles.colSpan]}>
                      withheld — group below the reporting threshold
                    </Text>
                  ) : (
                    <>
                      <Text style={[styles.tableCell, styles.colNumber]}>
                        {thousands(row.eligible)}
                      </Text>
                      <Text style={[styles.tableCell, styles.colNumber]}>
                        {thousands(row.ballots)}
                      </Text>
                      <Text style={[styles.tableCell, styles.colNumber]}>{percent(row.percent)}</Text>
                    </>
                  )}
                </View>
              ))}
              <View style={[styles.tableRow, styles.tableTotal]} wrap={false}>
                <Text style={[styles.tableTotalCell, styles.colFaculty]}>All faculties</Text>
                <Text style={[styles.tableTotalCell, styles.colNumber]}>
                  {thousands(snapshot.eligible_voters)}
                </Text>
                <Text style={[styles.tableTotalCell, styles.colNumber]}>
                  {thousands(snapshot.ballots_cast)}
                </Text>
                <Text style={[styles.tableTotalCell, styles.colNumber]}>
                  {percent(snapshot.turnout_percent)}
                </Text>
              </View>
            </>
          ) : null}

          <View style={styles.hashBox} wrap={false}>
            <Text style={styles.hashText}>Ballot snapshot SHA-256 • {snapshot.content_hash}</Text>
          </View>

          <View style={styles.signatures} wrap={false}>
            <View style={styles.signature}>
              <Text style={styles.signatureRole}>RETURNING OFFICER</Text>
              <Text style={styles.signatureName}> </Text>
              <View style={styles.signatureRule} />
              <Text style={styles.signatureHint}>signature and date</Text>
            </View>
            <View style={styles.signature}>
              <Text style={styles.signatureRole}>CHAIR, ELECTORAL COMMISSION</Text>
              <Text style={styles.signatureName}> </Text>
              <View style={styles.signatureRule} />
              <Text style={styles.signatureHint}>signature and date</Text>
            </View>
          </View>

          <Text style={[styles.footnote, { marginTop: 14 }]}>
            {certified
              ? `Certified ${stamp(snapshot.certified_at)}. This document reports aggregate counts only; no ballot can be traced to a voter.`
              : `Provisional count computed ${dayStamp(snapshot.computed_at)}. The leading candidate in each position is not yet an elected officer.`}
          </Text>
        </View>

        <View style={styles.footer} fixed>
          <View style={styles.footerGroup}>
            <Text style={styles.footerText}>campVote</Text>
            <Text style={styles.footerSep}>•</Text>
            <Text style={styles.footerText}>{election.id}</Text>
            <Text style={styles.footerSep}>•</Text>
            <Text style={styles.footerText}>{election.title}</Text>
          </View>
          <Text
            style={styles.footerText}
            render={({ pageNumber, totalPages }) => `Page ${pageNumber} of ${totalPages}`}
          />
        </View>
      </Page>
    </Document>
  );
}
