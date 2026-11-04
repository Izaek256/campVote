"use client";

/**
 * The one-page election notice for boards, emails and the election page.
 *
 * It carries only configuration that is already public to every signed-in
 * student — the window, who may vote, the result policy, the approved
 * candidates and how to vote. Never a ballot choice, never a voter.
 *
 * Sections that have no data are left out rather than padded, so an election
 * with no approved candidates yet still prints a clean sheet.
 */
import { Document, Page, StyleSheet, Text, View } from "@react-pdf/renderer";
import { categoryLabel, statusLabel } from "../format";
import type { BallotContent, Election } from "../types";
import {
  dayStamp,
  INK,
  INK_2,
  INK_3,
  LINE,
  MARGIN_X,
  ON_INK,
  ON_INK_DIM,
  ON_INK_LINE,
  PRIMARY,
  SUNKEN,
  thousands,
  WHITE,
} from "./theme";

interface Props {
  election: Election;
  /** Positions with their approved candidates, as a voter may read them. */
  positions: BallotContent["positions"];
}

const styles = StyleSheet.create({
  page: {
    backgroundColor: WHITE,
    color: INK,
    fontFamily: "Helvetica",
    fontSize: 9,
    paddingBottom: 52,
  },

  band: { backgroundColor: INK, paddingHorizontal: MARGIN_X, paddingVertical: 22 },
  bandRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  wordmark: { color: WHITE, fontSize: 12, fontFamily: "Helvetica-Bold", letterSpacing: 0.3 },
  badge: { borderWidth: 1, borderColor: ON_INK_LINE, paddingVertical: 3, paddingHorizontal: 8 },
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

  sectionHead: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-end",
    marginTop: 24,
    marginBottom: 4,
    borderBottomWidth: 1,
    borderColor: LINE,
    paddingBottom: 6,
  },
  sectionTitle: { fontSize: 12, fontFamily: "Helvetica-Bold", color: INK },
  sectionNote: { fontSize: 6.5, letterSpacing: 1, color: INK_3 },

  factRow: {
    flexDirection: "row",
    borderBottomWidth: 1,
    borderColor: LINE,
    paddingVertical: 9,
  },
  factLabel: { width: 116, fontSize: 6.5, letterSpacing: 1.2, color: INK_3, paddingTop: 2 },
  factValue: { flex: 1, fontSize: 9.5, color: INK, lineHeight: 1.4 },
  factHint: { fontSize: 7.5, color: INK_3 },

  timelineRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderColor: LINE,
  },
  timelineLabel: { fontSize: 9, color: INK },
  timelineValue: { fontSize: 9, color: INK_3 },

  tableHead: {
    flexDirection: "row",
    backgroundColor: SUNKEN,
    paddingVertical: 7,
    paddingHorizontal: 8,
  },
  tableHeadCell: { fontSize: 6.5, letterSpacing: 1, color: INK_3 },
  tableRow: {
    flexDirection: "row",
    paddingVertical: 7,
    paddingHorizontal: 8,
    borderBottomWidth: 1,
    borderColor: LINE,
  },
  tableCell: { fontSize: 8.5, color: INK },
  colName: { flex: 3 },
  colBallot: { flex: 1.1 },
  colPosition: { flex: 3 },
  colStatus: { flex: 1.4 },

  steps: { flexDirection: "row", marginTop: 12 },
  step: { flex: 1, paddingRight: 10 },
  stepIndex: { fontSize: 13, fontFamily: "Helvetica-Bold", color: PRIMARY },
  stepTitle: { fontSize: 9, fontFamily: "Helvetica-Bold", color: INK, marginTop: 5 },
  stepBody: { fontSize: 7.5, color: INK_3, marginTop: 3, lineHeight: 1.35 },
  stepTime: { fontSize: 7.5, color: PRIMARY, marginTop: 10 },

  notice: {
    marginTop: 22,
    borderWidth: 1,
    borderColor: LINE,
    borderLeftWidth: 3,
    borderLeftColor: PRIMARY,
    backgroundColor: SUNKEN,
    paddingVertical: 12,
    paddingHorizontal: 14,
  },
  noticeTitle: { fontSize: 10, fontFamily: "Helvetica-Bold", color: INK },
  noticeBody: { fontSize: 8, color: INK_2, marginTop: 5, lineHeight: 1.45 },

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

function resultPolicyText(policy: Election["result_policy"]): string {
  switch (policy) {
    case "LIVE":
      return "Shown live while voting is open, and kept after it closes.";
    case "CERTIFIED_ONLY":
      return "Published only after the returning officer certifies the count.";
    default:
      return "Published automatically after voting closes. No results are shown while voting is open.";
  }
}

function resultPolicyShort(policy: Election["result_policy"]): string {
  switch (policy) {
    case "LIVE":
      return "Live during voting";
    case "CERTIFIED_ONLY":
      return "After certification";
    default:
      return "After voting closes";
  }
}

/** Who the roster rule admits, in plain words. A blank rule means everyone. */
function whoCanVote(election: Election): string {
  const parts: string[] = [];
  const domain = (election.allowed_domain ?? "").trim();
  parts.push(domain ? `Students with a ${domain} address` : "All enrolled students");
  const years = (election.allowed_years ?? "").trim();
  if (years) parts.push(`Year ${years}`);
  const courses = (election.allowed_courses ?? "").trim();
  if (courses) parts.push(courses);
  const eligible = election.eligibility_frozen_count || 0;
  parts.push(`${thousands(eligible)} eligible`);
  return parts.join(" • ");
}

export function ElectionInfoSheetDocument({ election, positions }: Props) {
  const rows = positions.flatMap((position) =>
    position.candidates.map((candidate) => ({
      id: candidate.id,
      name: candidate.full_name,
      ballot: `#${candidate.ballot_number}`,
      position: position.title,
    }))
  );
  const approved = rows.length;
  const applicationsClose = "When voting opens";

  return (
    <Document
      title={`Information sheet • ${election.title}`}
      author="campVote Electoral Commission"
      subject={`Election ${election.id}`}
      creator="campVote"
    >
      <Page size="A4" style={styles.page} wrap>
        <View style={styles.band}>
          <View style={styles.bandRow}>
            <Text style={styles.wordmark}>campVote</Text>
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{statusLabel(election.status).toUpperCase()}</Text>
            </View>
          </View>
          <Text style={styles.eyebrow}>ELECTION INFORMATION SHEET</Text>
          <Text style={styles.bandTitle}>{election.title}</Text>
          <Text style={styles.bandMeta}>
            {categoryLabel(election.category)} • {election.organizing_unit} • {election.id}
          </Text>
        </View>

        <View style={styles.body}>
          <View style={styles.sectionHead}>
            <Text style={styles.sectionTitle}>Key facts</Text>
            <Text style={styles.sectionNote}>{election.id}</Text>
          </View>

          <View style={styles.factRow}>
            <Text style={styles.factLabel}>VOTING OPENS</Text>
            <Text style={styles.factValue}>{dayStamp(election.starts_at)}</Text>
          </View>
          <View style={styles.factRow}>
            <Text style={styles.factLabel}>VOTING CLOSES</Text>
            <Text style={styles.factValue}>
              {dayStamp(election.ends_at)}{" "}
              <Text style={styles.factHint}>(server clock decides)</Text>
            </Text>
          </View>
          <View style={styles.factRow}>
            <Text style={styles.factLabel}>WHO CAN VOTE</Text>
            <Text style={styles.factValue}>{whoCanVote(election)}</Text>
          </View>
          <View style={styles.factRow}>
            <Text style={styles.factLabel}>RESULTS</Text>
            <Text style={styles.factValue}>{resultPolicyText(election.result_policy)}</Text>
          </View>

          <View style={styles.sectionHead}>
            <Text style={styles.sectionTitle}>Timeline</Text>
          </View>
          <View style={styles.timelineRow}>
            <Text style={styles.timelineLabel}>Candidate applications</Text>
            <Text style={styles.timelineValue}>{applicationsClose}</Text>
          </View>
          <View style={styles.timelineRow}>
            <Text style={styles.timelineLabel}>Voting</Text>
            <Text style={styles.timelineValue}>{dayStamp(election.starts_at)}</Text>
          </View>
          <View style={styles.timelineRow}>
            <Text style={styles.timelineLabel}>Results</Text>
            <Text style={styles.timelineValue}>{resultPolicyShort(election.result_policy)}</Text>
          </View>

          {approved > 0 ? (
            <>
              <View style={styles.sectionHead}>
                <Text style={styles.sectionTitle}>Approved candidates</Text>
                <Text style={styles.sectionNote}>
                  {approved} {approved === 1 ? "CANDIDATE" : "CANDIDATES"} • {positions.length}{" "}
                  {positions.length === 1 ? "POSITION" : "POSITIONS"}
                </Text>
              </View>
              <View style={styles.tableHead}>
                <Text style={[styles.tableHeadCell, styles.colName]}>CANDIDATE</Text>
                <Text style={[styles.tableHeadCell, styles.colBallot]}>BALLOT</Text>
                <Text style={[styles.tableHeadCell, styles.colPosition]}>POSITION</Text>
                <Text style={[styles.tableHeadCell, styles.colStatus]}>STATUS</Text>
              </View>
              {rows.map((row) => (
                <View key={row.id} style={styles.tableRow} wrap={false}>
                  <Text style={[styles.tableCell, styles.colName]}>{row.name}</Text>
                  <Text style={[styles.tableCell, styles.colBallot]}>{row.ballot}</Text>
                  <Text style={[styles.tableCell, styles.colPosition]}>{row.position}</Text>
                  <Text style={[styles.tableCell, styles.colStatus]}>Approved</Text>
                </View>
              ))}
            </>
          ) : null}

          <View style={styles.sectionHead}>
            <Text style={styles.sectionTitle}>How to vote</Text>
            <Text style={styles.sectionNote}>ABOUT 2 MINUTES</Text>
          </View>
          <View style={styles.steps}>
            <View style={styles.step}>
              <Text style={styles.stepIndex}>01</Text>
              <Text style={styles.stepTitle}>Sign in</Text>
              <Text style={styles.stepBody}>University email and the code we send you.</Text>
            </View>
            <View style={styles.step}>
              <Text style={styles.stepIndex}>02</Text>
              <Text style={styles.stepTitle}>Review</Text>
              <Text style={styles.stepBody}>Open the election and read the manifestos.</Text>
            </View>
            <View style={styles.step}>
              <Text style={styles.stepIndex}>03</Text>
              <Text style={styles.stepTitle}>Choose</Text>
              <Text style={styles.stepBody}>One selection per position. Do not leave blank.</Text>
            </View>
            <View style={styles.step}>
              <Text style={styles.stepIndex}>04</Text>
              <Text style={styles.stepTitle}>Confirm</Text>
              <Text style={styles.stepBody}>Review, then submit. You cannot change it after.</Text>
            </View>
          </View>

          <View style={styles.notice} wrap={false}>
            <Text style={styles.noticeTitle}>Your vote is secret</Text>
            <Text style={styles.noticeBody}>
              Officials and candidates cannot see your choices. Ballots are stored without a name or
              student number, so a vote can never be traced back to the person who cast it.
            </Text>
          </View>
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
