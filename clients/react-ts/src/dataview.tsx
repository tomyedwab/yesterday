import { useEffect, useState } from "react";

import { useConnectionState } from "./connection-state.tsx";
import { fetchAuthenticated } from "./auth.tsx";

const DataViewCache = new Map<string, any>();
const DataViewInFlightRequests = new Map<string, Promise<any>>();
const DataViewRefCounts = new Map<string, { count: number; expiry: number | null }>();

export default function useDataView(
  apiPath: string,
  apiParams: { [key: string]: string } = {},
): [boolean, any] {
  const connectionState = useConnectionState();
  const currentEventId = connectionState.latestEventId;
  const [response, setResponse] = useState(null);
  const [loading, setLoading] = useState(true);

  let queryKey = `${apiPath}@${currentEventId}`;
  for (const [key, value] of Object.entries(apiParams)) {
    queryKey += `&${key}=${value}`;
  }

  useEffect(() => {
    const counts = DataViewRefCounts.get(queryKey);
    if (!counts) {
      DataViewRefCounts.set(queryKey, {
        count: 1,
        expiry: null,
      });
    } else {
      DataViewRefCounts.set(queryKey, {
        count: counts.count + 1,
        expiry: counts.expiry,
      });
    }
    if (DataViewCache.get(queryKey)) {
      setResponse(DataViewCache.get(queryKey));
      setLoading(false);
      return;
    }
    const inFlightRequest = DataViewInFlightRequests.get(queryKey);
    if (inFlightRequest) {
      inFlightRequest.then((_: any) => {
        setResponse(DataViewCache.get(queryKey));
        setLoading(false);
      });
      return;
    }
    const fetchData = async () => {
      let encodedUrl = `${apiPath}?e=${currentEventId}`;
      for (const [key, value] of Object.entries(apiParams)) {
        encodedUrl += `&${key}=${encodeURIComponent(value)}`;
      }
      DataViewInFlightRequests.set(queryKey, new Promise(
        async (resolve, reject) => {
          let response: Response | null = null;
          try {
            response = await fetchAuthenticated(encodedUrl);
            const respData = await response.json();
            DataViewCache.set(queryKey, respData);
            DataViewInFlightRequests.delete(queryKey);
            setResponse(respData);
            setLoading(false);
            resolve(respData);

            const now = Date.now();
            DataViewRefCounts.forEach((refCount, key) => {
              if (
                refCount.count === 0 &&
                refCount.expiry !== null &&
                refCount.expiry < now
              ) {
                console.log(`Expiring query ${key}`);
                DataViewRefCounts.delete(key);
                DataViewCache.delete(key);
              }
            });
          } catch (e) {
            // TODO(tom) report error in UI
            console.log(
              `Error fetching query data for key ${queryKey}:`,
              response,
              e,
            );
            reject(e);
          }
        },
      ));
    };

    if (currentEventId > 0) {
      fetchData();
    }

    return () => {
      const refCount = DataViewRefCounts.get(queryKey);
      if (!refCount) {
        return;
      }
      refCount.count -= 1;
      if (refCount.count === 0) {
        // Set expiry to 5 minutes from now
        refCount.expiry = Date.now() + 300000;
      }
    };
  }, [currentEventId, queryKey]);

  return [loading, response];
}
