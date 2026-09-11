import { useEffect, useState } from "react";
import { alarmService, type Alarm } from "../services/api";

export function AlarmList() {
  const [alarms, setAlarms] = useState<Alarm[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchAlarms();
  }, []);

  const fetchAlarms = async () => {
    try {
      setLoading(true);
      const data = await alarmService.getAlarms();
      setAlarms(data);
      setError(null);
    } catch (err: any) {
      setError(err.message || "Erro ao carregar alarmes");
    } finally {
      setLoading(false);
    }
  };

  const handleClose = async (id: string) => {
    try {
      await alarmService.closeAlarm(id);
      // Atualiza o estado da lista localmente para refletir imediatamente na UI
      setAlarms((prevAlarms) =>
        prevAlarms.map((alarm) =>
          alarm.id === id ? { ...alarm, status: "CLOSED" } : alarm
        )
      );
    } catch (err: any) {
      alert(err.message || "Falha ao desligar o alarme.");
    }
  };

  // Cálculos diretos na renderização
  const openCount = alarms.filter((a) => a.status === "OPEN").length;
  const closedCount = alarms.length - openCount;

  return (
    <div className="border border-line bg-surface p-5 flex flex-col h-full min-h-[400px]">
      <div className="flex items-center justify-between border-b border-line pb-4 mb-4">
        <div>
          <h2 className="font-semibold text-lg tracking-tight">Alarmes</h2>
          <p className="text-sm text-muted mt-1">
            <span className="font-medium text-ink">{openCount} abertos</span> · {closedCount} fechados
          </p>
        </div>
        <button
          onClick={fetchAlarms}
          className="text-sm border border-line bg-panel px-3 py-1 hover:bg-line/20 transition-colors"
          disabled={loading}
        >
          {loading ? "Atualizando..." : "Atualizar"}
        </button>
      </div>

      {error ? (
        <div className="text-sm text-alarm bg-alarm-soft p-3 border border-alarm/20">
          {error}
        </div>
      ) : (
        <div className="flex flex-col gap-2 overflow-y-auto pr-2">
          {alarms.length === 0 && !loading && (
            <p className="text-sm text-muted text-center py-8">Nenhum evento registrado.</p>
          )}

          {alarms.map((alarm) => {
            const isOpen = alarm.status === "OPEN";

            return (
              <div
                key={alarm.id}
                className="flex items-center justify-between p-2 border-b border-line/50 last:border-0 hover:bg-panel transition-colors"
              >
                <div className="flex items-center gap-4">
                  {/* Indicador Luminoso */}
                  <div
                    className={`size-3 rounded-full shrink-0 ${
                      isOpen ? "bg-alarm animate-alarm" : "bg-ok"
                    }`}
                  />
                  
                  {/* Dados Alinhados */}
                  <div className="flex flex-col">
                    <span className="font-medium text-sm" data-numeric>
                      {alarm.device_id}
                    </span>
                    <span className="text-xs text-muted" data-numeric>
                      {new Date(alarm.created_at).toLocaleTimeString()}
                    </span>
                  </div>
                </div>

                {/* Ação / Status */}
                <div>
                  {isOpen ? (
                    <button
                      onClick={() => handleClose(alarm.id)}
                      className="text-xs font-medium bg-panel border border-line px-3 py-1.5 hover:border-ink focus:outline-none focus-visible transition-colors"
                    >
                      [Desligar]
                    </button>
                  ) : (
                    <span className="text-xs font-medium text-ok mr-2 tracking-wider">
                      CLOSED
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}