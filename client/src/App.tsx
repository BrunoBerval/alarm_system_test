import { SingleEventForm } from "./components/SingleEventForm";
import { BatchEventForm } from "./components/BatchEventForm";
import { AlarmList } from "./components/AlarmList";

export default function App() {
  return (
    <main className="mx-auto max-w-5xl px-6 py-12">
      <h1 className="text-3xl font-semibold tracking-tight mb-8">
        Painel de Operação
      </h1>

      {/* Grid simulando as placas de metal do painel */}
      <div className="grid grid-cols-1 md:grid-cols-[320px_1fr] gap-6 items-start">
        
        {/* Coluna Esquerda: Ações de Disparo */}
        <div className="flex flex-col gap-6">
          <SingleEventForm />
          <BatchEventForm />
        </div>

        {/* Coluna Direita: Monitoramento */}
        <div className="h-full">
          <AlarmList />
        </div>

      </div>
    </main>
  );
}