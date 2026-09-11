export interface Alarm {
  // UUID do alarme[cite: 2]
  id: string;
  device_id: string;
  status: 'OPEN' | 'CLOSED';
  created_at: string;
  closed_at?: string;
}

export interface EventPayload {
  // Identificador do sensor[cite: 1]
  device_id: string;
  // Tipo de evento[cite: 1]
  type: string;
}

// As tipagens do import.meta.env são nativas do Vite
const ALARM_API = import.meta.env.VITE_ALARM_API_URL as string;
const EVENT_API = import.meta.env.VITE_EVENT_API_URL as string;

export const alarmService = {
  async getAlarms(): Promise<Alarm[]> {
    const response = await fetch(`${ALARM_API}/alarms`);
    
    if (!response.ok) {
      throw new Error('Falha ao carregar a lista de alarmes');
    }
    
    return response.json();
  },

  async closeAlarm(id: string): Promise<{ status: string; id: string }> {
    // Altera o status do alarme para CLOSED[cite: 2]
    const response = await fetch(`${ALARM_API}/alarms/${id}/close`, {
      method: 'PATCH',
    });

    if (!response.ok) {
      throw new Error(`Erro ao desligar o alarme ${id}`);
    }

    return response.json();
  }
};

export const eventService = {
  async dispatchEvent(payload: EventPayload): Promise<{ status: string }> {
    // Assumindo rota '/' baseada no handler de eventos configurado no Go
    const response = await fetch(`${EVENT_API}/events`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    // O serviço retorna 503 caso o buffer de eventos esteja cheio e a requisição seja rejeitada por sobrecarga[cite: 1].
    if (response.status === 503) {
      throw new Error('Serviço sobrecarregado, tente novamente');
    }

    // A publicação acontece de forma assíncrona, retornando 202 para indicar que o evento foi aceito e enfileirado com sucesso[cite: 1].
    if (!response.ok && response.status !== 202) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || 'Erro ao disparar evento');
    }

    return response.json();
  }
};