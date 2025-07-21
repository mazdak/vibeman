import VibemanUI from "./components/VibemanUI";
import { QueryProvider } from "./providers/query-provider";

function App() {
  return (
    <QueryProvider>
      <VibemanUI />
    </QueryProvider>
  );
}

export default App;
