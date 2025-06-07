//import { useState } from "react";
import "./App.css";
import {
  ConnectionStateProvider,
  useConnectionState,
} from "@tomyedwab/yesterday";

function ShowConnectionState() {
  const connectionState = useConnectionState();
  return (
    <div>
      <p>{connectionState.serverVersion}</p>
      <p>{connectionState.connected ? "Connected" : "Disconnected"}</p>
    </div>
  );
}

function App() {
  return (
    <ConnectionStateProvider>
      <h1>Vite + React</h1>
      <p>Current state:</p>
      <ShowConnectionState />
    </ConnectionStateProvider>
  );
}

export default App;
