import { useState, type FormEvent } from "react";
import { eventService } from "../services/api";

export function SingleEventForm() {
  const [sensorId, setSensorId] = useState<string>("42");
  const [eventType, setEventType] = useState<string>("MOTION_DETECTED");
  const [loading, setLoading] = useState<boolean>(false);
  const [feedback, setFeedback] = useState<{ type: "success" | "error"; msg: string } | null>(null);

  // Padroniza a exibição do ID com 3 dígitos (ex: "sensor-042")
  const formattedDeviceId = `sensor-${String(sensorId).padStart(3, "0")}`;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!sensorId.trim()) return;

    setLoading(true);
    setFeedback(null);

    try {
      // O evento precisa obrigatoriamente de device_id e type
      await eventService.dispatchEvent({
        device_id: formattedDeviceId,
        type: eventType,
      });
      
      setFeedback({ type: "success", msg: "Evento disparado." });
    } catch (error: any) {
      setFeedback({ type: "error", msg: error.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <form 
      onSubmit={handleSubmit}
      className="border border-line bg-surface p-5 flex flex-col gap-4 max-w-sm"
    >
      <h2 className="font-semibold text-lg tracking-tight">Disparar evento</h2>

      <div className="flex flex-col gap-1">
        <label htmlFor="sensor" className="text-sm font-medium">Sensor</label>
        <input
          id="sensor"
          type="number"
          min="1"
          value={sensorId}
          onChange={(e) => setSensorId(e.target.value)}
          className="border border-line bg-panel p-2 text-ink focus:outline-none focus-visible"
          disabled={loading}
          required
        />
        <span className="text-xs text-muted" data-numeric>
          → {formattedDeviceId}
        </span>
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="type" className="text-sm font-medium">Tipo</label>
        <select
          id="type"
          value={eventType}
          onChange={(e) => setEventType(e.target.value)}
          className="border border-line bg-panel p-2 text-ink focus:outline-none focus-visible cursor-pointer"
          disabled={loading}
        >
          <option value="MOTION_DETECTED">MOTION_DETECTED</option>
          <option value="NULL">NULL</option>
        </select>
      </div>

      <button
        type="submit"
        disabled={loading}
        className="mt-2 bg-ink text-panel py-2 font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
      >
        {loading ? "Enviando..." : "Disparar evento"}
      </button>

      {feedback && (
        <p 
          className={`text-sm mt-2 font-medium ${
            feedback.type === "error" ? "text-alarm" : "text-ok"
          }`}
        >
          {feedback.type === "error" ? "⚠ " : "✓ "}{feedback.msg}
        </p>
      )}
    </form>
  );
}