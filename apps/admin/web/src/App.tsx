//import { useState } from "react";
import { Provider } from "./components/ui/provider";
import { Button } from "@chakra-ui/react";

import "./App.css";
import {
  ConnectionStateProvider,
  CreatePendingEvent,
  useConnectionState,
  useConnectionDispatch,
} from "@tomyedwab/yesterday";
import { useUsersView } from "./dataviews/users";

function ShowConnectionState() {
  const connectionState = useConnectionState();
  return (
    <div>
      <p>{connectionState.serverVersion}</p>
      <p>{connectionState.connected ? "Connected" : "Disconnected"}</p>
    </div>
  );
}

function ShowUsers() {
  const connectDispatch = useConnectionDispatch();
  const [loading, users] = useUsersView();
  if (loading) {
    return <div>Loading users...</div>;
  }
  const doAddUser = () => {
    connectDispatch(
      CreatePendingEvent("adduser:", {
        type: "AddUser",
        username: "new_user",
      }),
    );
  };
  return (
    <div>
      {users.map((user) => (
        <div>
          User {user.id}: {user.username}
        </div>
      ))}
      <Button onClick={doAddUser}>Add user</Button>
    </div>
  );
}

function App() {
  return (
    <Provider>
      <ConnectionStateProvider>
        <h1>Vite + React</h1>
        <p>Current state:</p>
        <ShowConnectionState />
        <ShowUsers />
      </ConnectionStateProvider>
    </Provider>
  );
}

export default App;
