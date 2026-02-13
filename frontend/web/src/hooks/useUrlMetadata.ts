import { useState, useCallback } from "react";
import { GetURLMetadataResponse } from "@/types/proto/api/v1/shortcut_service";

interface UseUrlMetadataReturn {
  fetchMetadata: (url: string) => Promise<GetURLMetadataResponse | null>;
  loading: boolean;
  error: string | null;
}

export function useUrlMetadata(): UseUrlMetadataReturn {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMetadata = useCallback(async (url: string): Promise<GetURLMetadataResponse | null> => {
    if (!url) {
      return null;
    }

    // Validate URL format
    try {
      new URL(url);
    } catch {
      setError("Invalid URL format");
      return null;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/api/v1/shortcuts:fetchMetadata", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to fetch metadata (${response.status})`);
      }

      const data = await response.json();
      return data as GetURLMetadataResponse;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch URL metadata";
      setError(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return { fetchMetadata, loading, error };
}
