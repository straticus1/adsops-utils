#!/usr/bin/env python3
"""
Markdown Ticket Sync Service

Parses markdown tickets from various repositories and syncs them to the Changes
ticketing system. Designed for audit compliance (SOX, HIPAA, GDPR, GLBA).

Features:
- Parses markdown ticket files with various formats
- Maps to Changes ticket schema
- Deduplicates via tracking file
- Supports compliance frameworks
- Generates audit trail

Usage:
    python md-ticket-sync.py [--scan-dir /path/to/tickets] [--dry-run] [--submit]
    python md-ticket-sync.py --list-pending
    python md-ticket-sync.py --sync-all
"""

import os
import re
import json
import hashlib
import subprocess
import argparse
from pathlib import Path
from datetime import datetime
from typing import Optional, Dict, List, Any

# Configuration
CHANGES_CLI = Path(__file__).parent.parent / "changes"
SYNC_STATE_FILE = Path(__file__).parent.parent / "tickets" / ".md-sync-state.json"
DEFAULT_SCAN_DIRS = [
    Path.home() / "development" / "buildnft.xyz" / "docs" / "tickets",
    Path.home() / "development" / "ads-recovery" / "tickets",
    Path.home() / "development" / "afterdarksys.com" / "tickets",
    Path.home() / "development" / "dnsscience.io" / "tickets",
    Path.home() / "development" / "minttoken.xyz" / "tickets",
]

# Ticket parsing patterns
PATTERNS = {
    "title": re.compile(r"^#\s+(.+?)$", re.MULTILINE),
    "status": re.compile(r"\*?\*?Status\*?\*?:\s*(\w+)", re.IGNORECASE),
    "priority": re.compile(r"\*?\*?Priority\*?\*?:\s*(P[0-4]|Critical|High|Medium|Low|Urgent|Emergency|Normal)", re.IGNORECASE),
    "severity": re.compile(r"\*?\*?Severity\*?\*?:\s*(\w+)", re.IGNORECASE),
    "created": re.compile(r"\*?\*?Created\*?\*?:\s*(\d{4}-\d{2}-\d{2})", re.IGNORECASE),
    "resolved": re.compile(r"\*?\*?(?:Resolved|Completed)\*?\*?:\s*(\d{4}-\d{2}-\d{2})", re.IGNORECASE),
    "assignee": re.compile(r"\*?\*?(?:Assignee|Assigned)\*?\*?:\s*(.+?)(?:\n|$)", re.IGNORECASE),
    "category": re.compile(r"\*?\*?(?:Category|Component)\*?\*?:\s*(.+?)(?:\n|$)", re.IGNORECASE),
    "summary": re.compile(r"##\s+(?:Summary|Description)\s*\n+([\s\S]+?)(?=\n##|\n---|\Z)", re.IGNORECASE),
    "ticket_id": re.compile(r"^#\s+(\w+-\d+):", re.MULTILINE),
}

# Priority mapping
PRIORITY_MAP = {
    "p0": "emergency",
    "p1": "urgent",
    "p2": "high",
    "p3": "normal",
    "p4": "low",
    "critical": "emergency",
    "high": "high",
    "medium": "normal",
    "low": "low",
    "urgent": "urgent",
    "emergency": "emergency",
    "normal": "normal",
}

# Risk level mapping
RISK_MAP = {
    "p0": "critical",
    "p1": "high",
    "p2": "medium",
    "p3": "low",
    "p4": "low",
    "critical": "critical",
    "high": "high",
    "medium": "medium",
    "low": "low",
}


class TicketParser:
    """Parses markdown ticket files into structured data."""

    def __init__(self, filepath: Path):
        self.filepath = filepath
        self.content = filepath.read_text()
        self.data: Dict[str, Any] = {}

    def parse(self) -> Dict[str, Any]:
        """Parse the markdown file and extract ticket data."""
        self.data = {
            "source_file": str(self.filepath),
            "file_hash": self._compute_hash(),
            "parsed_at": datetime.utcnow().isoformat() + "Z",
        }

        # Extract title
        title_match = PATTERNS["title"].search(self.content)
        if title_match:
            self.data["title"] = title_match.group(1).strip()
            # Try to extract ticket ID from title
            id_match = re.match(r"(\w+-\d+)", self.data["title"])
            if id_match:
                self.data["original_id"] = id_match.group(1)

        # Extract other fields
        for field in ["status", "priority", "severity", "created", "resolved", "assignee", "category"]:
            match = PATTERNS[field].search(self.content)
            if match:
                self.data[field] = match.group(1).strip()

        # Extract summary/description
        summary_match = PATTERNS["summary"].search(self.content)
        if summary_match:
            self.data["description"] = summary_match.group(1).strip()
        else:
            # Fall back to first paragraph after title
            lines = self.content.split("\n")
            desc_lines = []
            in_desc = False
            for line in lines[1:]:  # Skip title
                if line.startswith("---"):
                    if in_desc:
                        break
                    in_desc = True
                    continue
                if in_desc and line.strip():
                    desc_lines.append(line)
                    if len(desc_lines) >= 5:
                        break
            self.data["description"] = "\n".join(desc_lines)

        # Determine compliance frameworks based on content
        self.data["compliance_frameworks"] = self._detect_compliance()

        # Determine affected systems
        self.data["affected_systems"] = self._detect_systems()

        # Map priority
        raw_priority = self.data.get("priority", self.data.get("severity", "normal")).lower()
        self.data["priority_mapped"] = PRIORITY_MAP.get(raw_priority, "normal")
        self.data["risk_mapped"] = RISK_MAP.get(raw_priority, "medium")

        # Determine ticket type
        self.data["type"] = self._detect_type()

        return self.data

    def _compute_hash(self) -> str:
        """Compute SHA256 hash of file content."""
        return hashlib.sha256(self.content.encode()).hexdigest()[:16]

    def _detect_compliance(self) -> List[str]:
        """Detect applicable compliance frameworks."""
        frameworks = []
        content_lower = self.content.lower()

        if any(term in content_lower for term in ["hipaa", "health", "medical", "patient"]):
            frameworks.append("hipaa")
        if any(term in content_lower for term in ["sox", "financial", "audit", "accounting"]):
            frameworks.append("sox")
        if any(term in content_lower for term in ["gdpr", "privacy", "personal data", "european"]):
            frameworks.append("gdpr")
        if any(term in content_lower for term in ["pci", "credit card", "payment"]):
            frameworks.append("pci_dss")
        if any(term in content_lower for term in ["security", "vulnerability", "exploit", "attack"]):
            frameworks.append("sox")  # Security issues affect financial audits
        if any(term in content_lower for term in ["blockchain", "crypto", "wallet", "token"]):
            frameworks.append("banking_secrecy_act")

        # Deduplicate
        return list(set(frameworks)) if frameworks else ["sox"]  # Default to SOX

    def _detect_systems(self) -> List[str]:
        """Detect affected systems from content."""
        systems = []
        content_lower = self.content.lower()

        # Map keywords to systems
        system_keywords = {
            "database": ["database", "postgresql", "mysql", "mongodb", "redis", "sql"],
            "api": ["api", "endpoint", "rest", "graphql"],
            "authentication": ["auth", "login", "session", "token", "jwt", "oauth"],
            "blockchain": ["blockchain", "ethereum", "solana", "contract", "web3"],
            "frontend": ["frontend", "ui", "react", "vue", "angular", "css"],
            "backend": ["backend", "server", "php", "python", "node", "flask"],
            "infrastructure": ["dns", "caddy", "nginx", "docker", "kubernetes", "oci", "aws"],
            "security": ["security", "vulnerability", "csrf", "xss", "injection"],
        }

        for system, keywords in system_keywords.items():
            if any(kw in content_lower for kw in keywords):
                systems.append(system)

        return systems if systems else ["general"]

    def _detect_type(self) -> str:
        """Detect ticket type from content."""
        content_lower = self.content.lower()
        title_lower = self.data.get("title", "").lower()

        if any(term in title_lower for term in ["security", "vulnerability", "cve"]):
            return "security"
        if any(term in title_lower for term in ["outage", "down", "incident"]):
            return "incident"
        if any(term in title_lower for term in ["bug", "fix", "error"]):
            return "bug_fix"
        if any(term in title_lower for term in ["feature", "implement", "add"]):
            return "enhancement"
        if any(term in content_lower for term in ["breaking change", "migration"]):
            return "breaking_change"

        return "standard"


class SyncState:
    """Manages sync state to avoid duplicate submissions."""

    def __init__(self, state_file: Path = SYNC_STATE_FILE):
        self.state_file = state_file
        self.state = self._load()

    def _load(self) -> Dict[str, Any]:
        """Load sync state from file."""
        if self.state_file.exists():
            return json.loads(self.state_file.read_text())
        return {"synced_tickets": {}, "last_sync": None}

    def save(self):
        """Save sync state to file."""
        self.state_file.parent.mkdir(parents=True, exist_ok=True)
        self.state_file.write_text(json.dumps(self.state, indent=2))

    def is_synced(self, source_file: str, file_hash: str) -> bool:
        """Check if a file has already been synced with same content."""
        synced = self.state["synced_tickets"].get(source_file, {})
        return synced.get("hash") == file_hash

    def mark_synced(self, source_file: str, file_hash: str, changes_id: str):
        """Mark a file as synced."""
        self.state["synced_tickets"][source_file] = {
            "hash": file_hash,
            "changes_id": changes_id,
            "synced_at": datetime.utcnow().isoformat() + "Z",
        }
        self.state["last_sync"] = datetime.utcnow().isoformat() + "Z"
        self.save()

    def get_changes_id(self, source_file: str) -> Optional[str]:
        """Get the Changes ticket ID for a source file."""
        return self.state["synced_tickets"].get(source_file, {}).get("changes_id")


class ChangesSubmitter:
    """Submits tickets to the Changes system via CLI."""

    def __init__(self, cli_path: Path = CHANGES_CLI, dry_run: bool = False):
        self.cli_path = cli_path
        self.dry_run = dry_run

        if not cli_path.exists():
            raise FileNotFoundError(f"Changes CLI not found at {cli_path}")

    def submit(self, ticket_data: Dict[str, Any]) -> Optional[str]:
        """Submit a ticket to Changes and return the ticket ID."""
        cmd = [
            str(self.cli_path), "ticket", "create",
            "--interactive=false",
            "--title", ticket_data.get("title", "Untitled Ticket"),
            "--description", self._build_description(ticket_data),
            "--priority", ticket_data.get("priority_mapped", "normal"),
            "--risk", ticket_data.get("risk_mapped", "medium"),
            "--change-type", ticket_data.get("type", "standard"),
        ]

        # Add compliance frameworks
        if ticket_data.get("compliance_frameworks"):
            cmd.extend(["--compliance", ",".join(ticket_data["compliance_frameworks"])])

        # Add affected systems
        if ticket_data.get("affected_systems"):
            cmd.extend(["--affected-systems", ",".join(ticket_data["affected_systems"])])

        # Add submit flag to move past draft
        cmd.append("--submit")

        if self.dry_run:
            print(f"[DRY RUN] Would execute: {' '.join(cmd[:6])}...")
            return f"DRY-RUN-{datetime.utcnow().strftime('%Y%m%d%H%M%S')}"

        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            if result.returncode == 0:
                # Parse ticket ID from output
                id_match = re.search(r"(CHG-\d{4}-\d{5})", result.stdout)
                if id_match:
                    return id_match.group(1)
                print(f"Warning: Could not parse ticket ID from output: {result.stdout}")
            else:
                print(f"Error submitting ticket: {result.stderr}")
        except subprocess.TimeoutExpired:
            print("Error: CLI command timed out")
        except Exception as e:
            print(f"Error executing CLI: {e}")

        return None

    def _build_description(self, ticket_data: Dict[str, Any]) -> str:
        """Build a rich description for the Changes ticket."""
        parts = []

        # Original description
        if ticket_data.get("description"):
            parts.append(ticket_data["description"])

        # Metadata section
        parts.append("\n---\n**Synced from Markdown Ticket**")

        if ticket_data.get("original_id"):
            parts.append(f"Original ID: {ticket_data['original_id']}")

        parts.append(f"Source: {ticket_data.get('source_file', 'unknown')}")

        if ticket_data.get("created"):
            parts.append(f"Originally Created: {ticket_data['created']}")

        if ticket_data.get("resolved"):
            parts.append(f"Originally Resolved: {ticket_data['resolved']}")

        if ticket_data.get("assignee"):
            parts.append(f"Original Assignee: {ticket_data['assignee']}")

        return "\n".join(parts)


def find_tickets(scan_dirs: List[Path]) -> List[Path]:
    """Find all markdown ticket files in scan directories."""
    tickets = []
    for scan_dir in scan_dirs:
        if scan_dir.exists():
            # Look for .md files in tickets directories
            for pattern in ["*.md", "**/tickets/*.md", "**/docs/tickets/*.md"]:
                tickets.extend(scan_dir.glob(pattern))

    # Deduplicate and sort
    return sorted(set(tickets))


def main():
    parser = argparse.ArgumentParser(description="Sync markdown tickets to Changes system")
    parser.add_argument("--scan-dir", type=Path, action="append", dest="scan_dirs",
                       help="Directory to scan for tickets (can specify multiple)")
    parser.add_argument("--dry-run", action="store_true",
                       help="Parse and show what would be synced without submitting")
    parser.add_argument("--submit", action="store_true",
                       help="Actually submit tickets to Changes system")
    parser.add_argument("--list-pending", action="store_true",
                       help="List tickets that haven't been synced yet")
    parser.add_argument("--sync-all", action="store_true",
                       help="Sync all pending tickets")
    parser.add_argument("--force", action="store_true",
                       help="Force re-sync even if already synced")
    args = parser.parse_args()

    # Determine scan directories
    scan_dirs = args.scan_dirs if args.scan_dirs else DEFAULT_SCAN_DIRS

    # Initialize state tracker
    state = SyncState()

    # Find all ticket files
    ticket_files = find_tickets(scan_dirs)
    print(f"Found {len(ticket_files)} markdown ticket files")

    pending = []
    synced = []

    for filepath in ticket_files:
        try:
            parser_obj = TicketParser(filepath)
            data = parser_obj.parse()

            if state.is_synced(str(filepath), data["file_hash"]) and not args.force:
                synced.append((filepath, state.get_changes_id(str(filepath))))
            else:
                pending.append((filepath, data))
        except Exception as e:
            print(f"Error parsing {filepath}: {e}")

    print(f"Already synced: {len(synced)}")
    print(f"Pending sync: {len(pending)}")

    if args.list_pending:
        print("\n=== Pending Tickets ===")
        for filepath, data in pending:
            print(f"\n{filepath.name}")
            print(f"  Title: {data.get('title', 'N/A')}")
            print(f"  Priority: {data.get('priority_mapped', 'normal')}")
            print(f"  Type: {data.get('type', 'standard')}")
            print(f"  Compliance: {', '.join(data.get('compliance_frameworks', []))}")
        return

    if args.sync_all or args.submit:
        if not pending:
            print("No pending tickets to sync")
            return

        submitter = ChangesSubmitter(dry_run=args.dry_run)

        print("\n=== Syncing Tickets ===")
        for filepath, data in pending:
            print(f"\nProcessing: {filepath.name}")
            print(f"  Title: {data.get('title', 'N/A')}")

            changes_id = submitter.submit(data)
            if changes_id:
                print(f"  -> Submitted as: {changes_id}")
                if not args.dry_run:
                    state.mark_synced(str(filepath), data["file_hash"], changes_id)
            else:
                print(f"  -> Failed to submit")

        print("\n=== Sync Complete ===")
        print(f"Processed {len(pending)} tickets")


if __name__ == "__main__":
    main()
