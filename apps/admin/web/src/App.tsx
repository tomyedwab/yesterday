import { Provider } from "./components/ui/provider";
import { ConnectionStateProvider } from "@tomyedwab/yesterday";
import { MainLayout } from "./components/layout/MainLayout";

function App() {
  return (
    <Provider>
      <ConnectionStateProvider>
        <MainLayout />
      </ConnectionStateProvider>
    </Provider>
  );
}

export default App;
