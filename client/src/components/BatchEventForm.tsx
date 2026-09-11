import { useState, type FormEvent } from "react";
import { eventService } from "../services/api";

export function BatchEventForm() {
  const [fromSensor, setFromSensor] = useState<number>(30);
  const [toSensor, setToSensor] = useState<number>(60);
  const [eventType, setEventType] = useState<string>("MOTION_DETECTED");
  const [loading, setLoading] = useState<boolean>(false);
  const [feedback, setFeedback] = useState<{ type: "success" | "error"; msg: string } | null>(null);

  const from = Number(fromSensor);
  const to = Number(toSensor);
  
  // Controle de estado direto, sem hooks de memoização
  const count = to >= from ? to - from + 1 : 0;
  const isInvalid = count <= 0;
  const isOverWarning = count > 20 && count <= 200;
  const isOverLimit = count > 200;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (isInvalid || isOverLimit || loading) return;

    setLoading(true);
    setFeedback(null);

    let successCount = 0;
    let errorCount = 0;

    // Disparo sequencial para mitigar a sobrecarga imediata do EventChannel no backend
    for (let i = from; i <= to; i++) {
      const formattedDeviceId = `sensor-${String(i).padStart(3, "0")}`;
      try {
        await eventService.dispatchEvent({
          device_id: formattedDeviceId,
          type: eventType,
        });
        successCount++;
      } catch (error) {
        errorCount++;
      }
    }

    if (errorCount === 0) {
      setFeedback({ type: "success", msg: `✓ ${successCount} eventos disparados.` });
    } else {
      setFeedback({ 
        type: "error", 
        msg: `⚠ ${successCount} disparados, ${errorCount} falharam (sobrecarga).` 
      });
    }

    setLoading(false);
  };

  return (
    <form 
      onSubmit={handleSubmit}
      className="border border-line bg-surface p-5 flex flex-col gap-4 max-w-sm"
    >
      <h2 className="font-semibold text-lg tracking-tight">Acionar vários</h2>

      <div className="flex flex-col gap-1">
        <label htmlFor="batchType" className="text-sm font-medium">Tipo</label>
        <select
          id="batchType"
          value={eventType}
          onChange={(e) => setEventType(e.target.value)}
          className="border border-line bg-panel p-2 text-ink focus:outline-none focus-visible cursor-pointer"
          disabled={loading}
        >
          <option value="MOTION_DETECTED">MOTION_DETECTED</option>
          <option value="NULL">NULL</option>
        </select>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex flex-col gap-1 w-full">
          <label htmlFor="fromSensor" className="text-sm font-medium">De</label>
          <input
            id="fromSensor"
            type="number"
            min="1"
            value={fromSensor}
            onChange={(e) => setFromSensor(Number(e.target.value))}
            className="border border-line bg-panel p-2 text-ink focus:outline-none focus-visible w-full"
            disabled={loading}
            required
          />
        </div>

        <div className="flex flex-col gap-1 w-full">
          <label htmlFor="toSensor" className="text-sm font-medium">Até</label>
          <input
            id="toSensor"
            type="number"
            min="1"
            value={toSensor}
            onChange={(e) => setToSensor(Number(e.target.value))}
            className="border border-line bg-panel p-2 text-ink focus:outline-none focus-visible w-full"
            disabled={loading}
            required
          />
        </div>
      </div>

      <div className="min-h-[2rem]">
        {isOverLimit && (
          <div className="text-sm font-medium text-info bg-alarm-soft p-2 border border-alarm/20">
            ⚠ Limite excedido. Máximo de 200 por lote.
          </div>
        )}
        {isOverWarning && !isOverLimit && (
          <div className="text-sm font-medium text-info p-2">
            ⚠ Aviso: {count} eventos serão enfileirados o sistema pode não aguentar.
          </div>
        )}
      </div>

      <button
        type="submit"
        disabled={loading || isInvalid || isOverLimit}
        className="mt-2 bg-ink text-panel py-2 font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
      >
        {loading ? "Enviando lote..." : `Disparar ${count > 0 ? count : ''} evts`}
      </button>

      {feedback && (
        <p 
          className={`text-sm mt-2 font-medium ${
            feedback.type === "error" ? "text-info" : "text-ok"
          }`}
        >
          {feedback.msg}
        </p>
      )}
    </form>
  );
}