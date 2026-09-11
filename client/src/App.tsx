/*
  Placeholder do scaffold.

  Serve para provar que o Tailwind, os tokens e a fonte estao carregando.
  Os componentes reais (StatusBar, SingleEventForm, RangeEventForm,
  AlarmList) entram nos proximos commits.
*/
export default function App() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight">
        Central de alarmes
      </h1>
      <p className="mt-2 max-w-prose text-muted">
        Dispare eventos de sensores e acompanhe os alarmes gerados.
      </p>

      <div className="mt-10 border border-line bg-surface p-5">
        <p className="text-sm text-muted">
          Scaffold ativo. Tokens, fonte e Tailwind carregados.
        </p>

        <div className="mt-4 flex items-center gap-6 text-sm">
          <span className="flex items-center gap-2">
            <span className="size-2.5 rounded-full bg-alarm animate-alarm" />
            alarme aberto
          </span>
          <span className="flex items-center gap-2">
            <span className="size-2.5 rounded-full bg-ok" />
            servico saudavel
          </span>
          <span data-numeric className="text-muted">
            sensor-042
          </span>
        </div>
      </div>
    </main>
  );
}
