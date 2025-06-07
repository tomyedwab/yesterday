import { useEffect, useState } from "react";

import { useConnectionState } from "./connection-state.tsx";
import { fetchAuthenticated } from "./auth.tsx";

const DataViewCache = new Map<string, any>();
const DataViewInFlightRequests = new Map<string, Promise<any>>();
const DataViewRefCounts = {};

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
    if (!DataViewRefCounts[queryKey]) {
      DataViewRefCounts[queryKey] = {
        count: 1,
        expiry: null,
      };
    } else {
      DataViewRefCounts[queryKey].count += 1;
    }
    if (DataViewCache[queryKey]) {
      setResponse(DataViewCache[queryKey]);
      setLoading(false);
      return;
    }
    if (DataViewInFlightRequests[queryKey]) {
      DataViewInFlightRequests[queryKey].then((_: any) => {
        setResponse(DataViewCache[queryKey]);
        setLoading(false);
      });
      return;
    }
    const fetchData = async () => {
      let encodedUrl = `${apiPath}?e=${currentEventId}`;
      for (const [key, value] of Object.entries(apiParams)) {
        encodedUrl += `&${key}=${encodeURIComponent(value)}`;
      }
      DataViewInFlightRequests[queryKey] = new Promise(
        async (resolve, reject) => {
          let response: Response | null = null;
          try {
            response = await fetchAuthenticated(encodedUrl);
            const respData = await response.json();
            DataViewCache[queryKey] = respData;
            delete DataViewInFlightRequests[queryKey];
            setResponse(respData);
            setLoading(false);
            resolve(respData);

            const now = Date.now();
            Object.keys(DataViewRefCounts).forEach((key) => {
              if (
                DataViewRefCounts[key].count === 0 &&
                DataViewRefCounts[key].expiry < now
              ) {
                console.log(`Expiring query ${key}`);
                delete DataViewRefCounts[key];
                delete DataViewCache[key];
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
      );
    };

    if (currentEventId > 0) {
      fetchData();
    }

    return () => {
      DataViewRefCounts[queryKey].count -= 1;
      if (DataViewRefCounts[queryKey].count === 0) {
        // Set expiry to 5 minutes from now
        DataViewRefCounts[queryKey].expiry = Date.now() + 300000;
      }
    };
  }, [currentEventId, queryKey]);

  return [loading, response];
}
