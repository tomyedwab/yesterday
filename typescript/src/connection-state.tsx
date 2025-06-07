// A reducer that tracks the current connection state, which tracks the current
// latest event ID if the connection is active as well as the state of saving
// new events.

import {
  createContext,
  useContext,
  useReducer,
  useEffect,
  type ReactNode,
} from "react";
import { fetchAuthenticated } from "./auth";

// Define the actions the dispatcher will accept and their parameters

export enum ConnectionActionTypes {
  ConnectionEstablished,
  ConnectionLost,
  ConnectionAddPendingEvent,
  ConnectionEventSaved,
  ConnectionEventSaveFailed,
}

export type ConnectionEstablished = {
  type: ConnectionActionTypes.ConnectionEstablished;
  latestEventId: number;
  serverVersion: string;
};

export type ConnectionLost = {
  type: ConnectionActionTypes.ConnectionLost;
};

export type ConnectionAddPendingEvent = {
  type: ConnectionActionTypes.ConnectionAddPendingEvent;
  pendingEvent: {
    clientId: string;
    event: any;
  };
};

export type ConnectionEventSaved = {
  type: ConnectionActionTypes.ConnectionEventSaved;
  latestEventId: number;
};

export type ConnectionEventSaveFailed = {
  type: ConnectionActionTypes.ConnectionEventSaveFailed;
};

export type ConnectionAction =
  | ConnectionEstablished
  | ConnectionLost
  | ConnectionAddPendingEvent
  | ConnectionEventSaved
  | ConnectionEventSaveFailed;

// The reducer state
export type ConnectionState = {
  connected: boolean;
  serverVersion: string;
  latestEventId: number;
  retryNumber: number;
  backoffSeconds: number;
  pollNumber: number;
  pendingEvent: {
    clientId: string;
    event: any;
  } | null;
  saveRetryNumber: number;
  saveBackoffSeconds: number;
};

// The reducer itself
export function ConnectionStateReducer(
  state: ConnectionState,
  action: ConnectionAction,
): ConnectionState {
  switch (action.type) {
    case ConnectionActionTypes.ConnectionEstablished:
      return {
        ...state,
        connected: true,
        serverVersion: action.serverVersion,
        retryNumber: 0,
        backoffSeconds: 1,
        latestEventId: action.latestEventId,
        pollNumber: state.pollNumber + 1,
      };
    case ConnectionActionTypes.ConnectionLost:
      return {
        ...state,
        connected: false,
        retryNumber: state.retryNumber + 1,
        backoffSeconds: Math.min(64, state.backoffSeconds * 2),
        latestEventId: 0,
        pollNumber: 0,
      };
    case ConnectionActionTypes.ConnectionAddPendingEvent:
      if (state.pendingEvent !== null) {
        // We're already saving an event, so we can't save another
        return state;
      }
      return {
        ...state,
        pendingEvent: action.pendingEvent,
        saveRetryNumber: 0,
        saveBackoffSeconds: 0,
      };
    case ConnectionActionTypes.ConnectionEventSaved:
      return {
        ...state,
        connected: true,
        retryNumber: 0,
        backoffSeconds: 1,
        latestEventId: Math.max(state.latestEventId, action.latestEventId),
        pollNumber: state.pollNumber + 1,
        pendingEvent: null,
        saveRetryNumber: 0,
        saveBackoffSeconds: 0,
      };
    case ConnectionActionTypes.ConnectionEventSaveFailed:
      return {
        ...state,
        saveRetryNumber: state.saveRetryNumber + 1,
        saveBackoffSeconds: Math.min(
          64,
          Math.max(1, state.saveBackoffSeconds) * 2,
        ),
      };
  }
}
const connectionDispatchContext = createContext<any>(
  (_: ConnectionAction) => {},
);

export const connectionStateContext = createContext<ConnectionState>({
  connected: false,
  latestEventId: 0,
  serverVersion: "",
  retryNumber: 0,
  backoffSeconds: 1,
  pollNumber: 0,
  pendingEvent: null,
  saveRetryNumber: 0,
  saveBackoffSeconds: 0,
});

let currentRequest: Promise<void> | null = null;

function ConnectionHandler({
  connectionState,
  connectionDispatch,
}: {
  connectionState: ConnectionState;
  connectionDispatch: (_: ConnectionAction) => void;
}) {
  // Background long polling for new events which also determines whethe we
  // are in a "connected" state.
  useEffect(() => {
    const fetchData = async () => {
      let response: Response | null = null;
      try {
        await new Promise((x) =>
          setTimeout(x, 1000 * connectionState.backoffSeconds),
        );
        response = await fetchAuthenticated(
          `/api/poll?e=${connectionState.latestEventId + 1}`,
        );
        if (response.status === 304) {
          // No new events have been saved since the last poll.
          connectionDispatch({
            type: ConnectionActionTypes.ConnectionEstablished,
            latestEventId: connectionState.latestEventId,
            serverVersion: connectionState.serverVersion,
          });
        } else {
          // A new event has been saved since the last poll, and the
          // event ID is in the response.
          const respData = await response.json();
          connectionDispatch({
            type: ConnectionActionTypes.ConnectionEstablished,
            latestEventId: respData.id,
            serverVersion: respData.version,
          });
        }
      } catch (e) {
        console.log("Error polling for latest event:", response, e);
        connectionDispatch({
          type: ConnectionActionTypes.ConnectionLost,
        });
      }
      currentRequest = null;
    };

    if (!currentRequest) {
      currentRequest = fetchData();
    }
  }, [
    connectionState.connected,
    connectionState.latestEventId,
    connectionState.retryNumber,
    connectionState.backoffSeconds,
    connectionState.pollNumber,
  ]);

  // Background saving of new events.
  useEffect(() => {
    const saveEvent = async (clientId: string, event: any) => {
      let response: Response | null = null;
      try {
        //await new Promise(x => setTimeout(x, 30000)); // Uncomment for testing only
        await new Promise((x) =>
          setTimeout(x, 1000 * connectionState.saveBackoffSeconds),
        );
        response = await fetchAuthenticated(`/api/publish?cid=${clientId}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(event),
        });
        const respData = await response.json();
        console.log("Saved event:", respData);
        connectionDispatch({
          type: ConnectionActionTypes.ConnectionEventSaved,
          latestEventId: respData.id,
        });
      } catch (e) {
        console.log("Error saving pending event:", response, e);
        connectionDispatch({
          type: ConnectionActionTypes.ConnectionEventSaveFailed,
        });
      }
    };

    if (connectionState.pendingEvent !== null) {
      saveEvent(
        connectionState.pendingEvent.clientId,
        connectionState.pendingEvent.event,
      );
    }
  }, [
    connectionState.pendingEvent,
    connectionState.saveRetryNumber,
    connectionState.saveBackoffSeconds,
  ]);

  return <></>;
}

export function ConnectionStateProvider({ children }: { children: ReactNode }) {
  const [connectionState, connectionDispatch] = useReducer(
    ConnectionStateReducer,
    {
      connected: false,
      serverVersion: "",
      latestEventId: 0,
      retryNumber: 0,
      backoffSeconds: 1,
      pollNumber: 0,
      pendingEvent: null,
      saveRetryNumber: 0,
      saveBackoffSeconds: 0,
    },
  );
  return (
    <connectionDispatchContext.Provider value={connectionDispatch}>
      <connectionStateContext.Provider value={connectionState}>
        <ConnectionHandler
          connectionDispatch={connectionDispatch}
          connectionState={connectionState}
        />
        {children}
      </connectionStateContext.Provider>
    </connectionDispatchContext.Provider>
  );
}

export const useConnectionState = () => useContext(connectionStateContext);
export const useConnectionDispatch = () =>
  useContext(connectionDispatchContext);
