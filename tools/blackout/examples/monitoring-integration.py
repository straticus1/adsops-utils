#!/usr/bin/env python3
"""
Example: OCI Observability Integration with Blackout Tool

This script demonstrates how to integrate the blackout tool with
monitoring systems to suppress alerts during maintenance windows.

Usage:
    python monitoring-integration.py --host api-server-1
"""

import json
import sys
from pathlib import Path
from datetime import datetime, timezone
from typing import List, Dict, Optional


class BlackoutManager:
    """Manages blackout status for monitoring integration."""

    def __init__(self, blackout_file: str = "/var/lib/adsops/active-blackouts.json"):
        self.blackout_file = Path(blackout_file)
        self._blackouts: List[Dict] = []
        self._load_blackouts()

    def _load_blackouts(self):
        """Load active blackouts from JSON file."""
        if not self.blackout_file.exists():
            print(f"Warning: Blackout file not found: {self.blackout_file}", file=sys.stderr)
            self._blackouts = []
            return

        try:
            with open(self.blackout_file, 'r') as f:
                self._blackouts = json.load(f)
        except json.JSONDecodeError as e:
            print(f"Error: Failed to parse blackout file: {e}", file=sys.stderr)
            self._blackouts = []
        except Exception as e:
            print(f"Error: Failed to read blackout file: {e}", file=sys.stderr)
            self._blackouts = []

    def is_in_blackout(self, hostname: str) -> bool:
        """Check if a host is currently in blackout."""
        now = datetime.now(timezone.utc)

        for blackout in self._blackouts:
            if blackout.get("hostname") == hostname:
                # Check if blackout is still active
                end_time_str = blackout.get("end_time")
                if end_time_str:
                    try:
                        end_time = datetime.fromisoformat(end_time_str.replace('Z', '+00:00'))
                        if end_time > now:
                            return True
                    except ValueError:
                        continue

        return False

    def get_blackout_info(self, hostname: str) -> Optional[Dict]:
        """Get blackout information for a host."""
        for blackout in self._blackouts:
            if blackout.get("hostname") == hostname:
                return blackout
        return None

    def get_all_blackouts(self) -> List[Dict]:
        """Get all active blackouts."""
        return self._blackouts


def should_check_host(hostname: str) -> bool:
    """
    Determine if a host should be checked by monitoring.

    Returns:
        True if host should be checked (no blackout)
        False if host is in blackout (skip checks)
    """
    manager = BlackoutManager()

    if manager.is_in_blackout(hostname):
        info = manager.get_blackout_info(hostname)
        print(f"⚠️  Host {hostname} is in maintenance blackout")
        print(f"   Ticket: {info['ticket']}")
        print(f"   Reason: {info['reason']}")
        print(f"   Ends: {info['end_time']}")
        print(f"   Remaining: {info.get('remaining_time', 'unknown')}")
        return False

    return True


def check_and_alert(hostname: str, check_func, alert_func):
    """
    Wrapper function that checks blackout status before running checks.

    Args:
        hostname: Hostname to check
        check_func: Function that performs the health check
        alert_func: Function that sends alerts

    Example:
        def check_http():
            return requests.get(f"http://{hostname}").status_code == 200

        def send_alert(message):
            pagerduty.trigger(message)

        check_and_alert("api-server-1", check_http, send_alert)
    """
    manager = BlackoutManager()

    # Skip checks if host is in blackout
    if manager.is_in_blackout(hostname):
        info = manager.get_blackout_info(hostname)
        print(f"[BLACKOUT] Skipping checks for {hostname} - {info['reason']}")
        return

    # Perform check
    try:
        result = check_func()
        if not result:
            # Check failed, send alert
            alert_func(f"Check failed for {hostname}")
    except Exception as e:
        alert_func(f"Check error for {hostname}: {e}")


def list_blackouts():
    """List all active blackouts."""
    manager = BlackoutManager()
    blackouts = manager.get_all_blackouts()

    if not blackouts:
        print("No active blackouts")
        return

    print(f"\n{'HOSTNAME':<20} {'TICKET':<15} {'ENDS':<25} {'REMAINING':<15}")
    print("-" * 80)

    for b in blackouts:
        print(f"{b['hostname']:<20} {b['ticket']:<15} {b['end_time']:<25} {b.get('remaining_time', 'unknown'):<15}")

    print(f"\nTotal: {len(blackouts)} active blackout(s)")


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(description="Blackout monitoring integration example")
    parser.add_argument("--host", help="Check if host is in blackout")
    parser.add_argument("--list", action="store_true", help="List all active blackouts")
    parser.add_argument("--file", default="/var/lib/adsops/active-blackouts.json",
                        help="Blackout JSON file path")

    args = parser.parse_args()

    if args.list:
        list_blackouts()
        return

    if args.host:
        if should_check_host(args.host):
            print(f"✅ Host {args.host} should be checked (not in blackout)")
            sys.exit(0)
        else:
            print(f"⚠️  Host {args.host} is in blackout (skip checks)")
            sys.exit(1)

    parser.print_help()


if __name__ == "__main__":
    main()
