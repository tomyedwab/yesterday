import { Provider } from "./components/ui/provider";
import { ConnectionStateProvider } from "@tomyedwab/yesterday";
import { MainLayout } from "./components/layout/MainLayout";
import { Toaster } from "./components/ui/toaster";

function App() {
  return (
    <Provider>
      <ConnectionStateProvider>
        <MainLayout />
        <Toaster />
      </ConnectionStateProvider>
    </Provider>
  );
}

export default App;
