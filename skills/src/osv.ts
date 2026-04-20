const OSV_QUERY_URL = 'https://api.osv.dev/v1/query';

export type OSVSeverity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'UNKNOWN';

export interface OSVAdvisory {
  id: string;
  summary?: string;
  severity: OSVSeverity;
}

export interface OSVResult {
  count: number;
  maxSeverity: OSVSeverity | null;
  advisories: OSVAdvisory[];
}

interface OSVVuln {
  id: string;
  summary?: string;
  severity?: Array<{ type: string; score: string }>;
  database_specific?: { severity?: string };
}

interface OSVQueryResponse {
  vulns?: OSVVuln[];
}

function cvssScoreToSeverity(score: string): OSVSeverity {
  // Extract base score from CVSS vector string (e.g. "CVSS:3.1/.../AV:N/.../")
  // or plain numeric score
  const numeric = parseFloat(score);
  if (!isNaN(numeric)) {
    if (numeric >= 9.0) return 'CRITICAL';
    if (numeric >= 7.0) return 'HIGH';
    if (numeric >= 4.0) return 'MEDIUM';
    if (numeric >= 0.1) return 'LOW';
    return 'UNKNOWN';
  }
  // CVSS vector string — extract the score differently; treat as UNKNOWN
  return 'UNKNOWN';
}

function parseSeverity(vuln: OSVVuln): OSVSeverity {
  // Prefer explicit database_specific severity string
  const dbSev = vuln.database_specific?.severity?.toUpperCase();
  if (dbSev === 'CRITICAL' || dbSev === 'HIGH' || dbSev === 'MEDIUM' || dbSev === 'LOW') {
    return dbSev;
  }

  // Fall back to CVSS score parsing
  for (const s of vuln.severity ?? []) {
    if (s.type === 'CVSS_V3' || s.type === 'CVSS_V2') {
      return cvssScoreToSeverity(s.score);
    }
  }

  return 'UNKNOWN';
}

const SEVERITY_RANK: Record<OSVSeverity, number> = {
  CRITICAL: 4,
  HIGH: 3,
  MEDIUM: 2,
  LOW: 1,
  UNKNOWN: 0,
};

function maxSeverity(severities: OSVSeverity[]): OSVSeverity | null {
  if (severities.length === 0) return null;
  return severities.reduce((a, b) => (SEVERITY_RANK[a] >= SEVERITY_RANK[b] ? a : b));
}

/**
 * Query the OSV database for known advisories affecting a GitHub repo.
 * Uses the pkg:github PURL which covers GitHub Actions and similar ecosystems.
 * Returns null on any error or timeout — never blocks installation.
 */
export async function fetchOSVAdvisories(
  ownerRepo: string,
  timeoutMs = 3000
): Promise<OSVResult | null> {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);

    const response = await fetch(OSV_QUERY_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        package: { purl: `pkg:github/${ownerRepo}` },
      }),
      signal: controller.signal,
    });
    clearTimeout(timeout);

    if (!response.ok) return null;

    const data = (await response.json()) as OSVQueryResponse;
    const vulns = data.vulns ?? [];

    const advisories: OSVAdvisory[] = vulns.map((v) => ({
      id: v.id,
      summary: v.summary,
      severity: parseSeverity(v),
    }));

    return {
      count: advisories.length,
      maxSeverity: maxSeverity(advisories.map((a) => a.severity)),
      advisories,
    };
  } catch {
    return null;
  }
}
