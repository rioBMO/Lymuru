"use client";

import { useState } from "react";

/**
 * Stub hook — provides the track availability check API that the SpotiFLAC
 * App component expects. Returns no-ops and empty state. Will be replaced
 * with real implementation when availability checking is wired in.
 */
export function useAvailability() {
  const [checkingTrackId, setCheckingTrackId] = useState<string | null>(null);
  const [availabilityMap] = useState(() => new Map<string, any>());

  function clearAvailability() {
    /* noop — stub */
  }

  async function checkAvailability(_trackId: string) {
    /* noop — stub */
  }

  return {
    clearAvailability,
    checkingTrackId,
    availabilityMap,
    checkAvailability,
  };
}
