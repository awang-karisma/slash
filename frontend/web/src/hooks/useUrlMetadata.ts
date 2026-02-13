import { useCallback, useState } from "react";
import { toast } from "sonner";
import { GetURLMetadataResponse } from "@/types/proto/api/v1/shortcut_service";

interface UseUrlMetadataReturn {
  fetchMetadata: (url: string, options?: { signal?: AbortSignal }) => Promise<GetURLMetadataResponse | null>;
  loading: boolean;
  error: string | null;
}

export function useUrlMetadata(): UseUrlMetadataReturn {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMetadata = useCallback(async (url: string, options?: { signal?: AbortSignal }): Promise<GetURLMetadataResponse | null> => {
    if (!url) {
      return null;
    }

    // Validate URL format
    try {
      new URL(url);
    } catch {
      const errorMsg = "Invalid URL format";
      setError(errorMsg);
      toast.error(errorMsg);
      return null;
    }

    // Don't show loading toast, just update state
    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/api/v1/shortcuts:fetchMetadata", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ url }),
        signal: options?.signal,
      });

      if (!response.ok) {
        let errorMessage = `Failed to fetch metadata (${response.status})`;

        try {
          const errorData = await response.json();
          if (errorData.message) {
            errorMessage = errorData.message;
          }
        } catch {
          // Ignore JSON parse errors
        }

        setError(errorMessage);
        toast.error(errorMessage);
        return null;
      }

      const data = await response.json();
      return data as GetURLMetadataResponse;
    } catch (err) {
      // Handle abort errors gracefully
      if (err instanceof DOMException && err.name === "AbortError") {
        setLoading(false);
        return null;
      }

      const message = err instanceof Error ? err.message : "Failed to fetch URL metadata";
      setError(message);
      toast.error(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    fetchMetadata,
    loading,
    error,
  };
}
