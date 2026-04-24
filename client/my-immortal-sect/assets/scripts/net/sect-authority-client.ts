import type {
    SectAuthorityCommandEnvelope,
    SectAuthorityCommandResponse,
    SectAuthorityJoinResponse,
} from './sect-authority-contract';

const DEFAULT_AUTHORITY_BASE_URL = 'http://127.0.0.1:8787';
const DEFAULT_TIMEOUT_MS = 2500;

export class SectAuthorityClient {
    constructor(
        private readonly baseUrl: string = DEFAULT_AUTHORITY_BASE_URL,
        private readonly timeoutMs: number = DEFAULT_TIMEOUT_MS,
    ) {}

    async joinSect(userId: string, sessionId: string, sectId?: string | null): Promise<SectAuthorityJoinResponse> {
        return this.request('/v1/authority/sect/join', {
            method: 'POST',
            body: JSON.stringify({
                userId,
                sessionId,
                sectId: sectId ?? '',
            }),
        });
    }

    async executeCommand<TPayload>(
        userId: string,
        sectId: string,
        sessionId: string,
        command: SectAuthorityCommandEnvelope<TPayload>,
    ): Promise<SectAuthorityCommandResponse> {
        return this.request('/v1/authority/sect/command', {
            method: 'POST',
            body: JSON.stringify({
                userId,
                sectId,
                sessionId,
                command,
            }),
        });
    }

    private async request<TResponse>(path: string, init: RequestInit): Promise<TResponse> {
        const controller = new AbortController();
        const timeoutId = globalThis.setTimeout(() => controller.abort(), this.timeoutMs);

        try {
            const response = await fetch(`${this.baseUrl}${path}`, {
                ...init,
                headers: {
                    'Content-Type': 'application/json',
                    ...(init.headers ?? {}),
                },
                signal: controller.signal,
            });

            const payload = (await response.json()) as TResponse | { error?: string };
            if (!response.ok) {
                const message = 'error' in payload && payload.error ? payload.error : `sect_authority_request_failed:${response.status}`;
                throw new Error(message);
            }
            return payload as TResponse;
        } catch (error) {
            if (error instanceof Error && error.name === 'AbortError') {
                throw new Error('sect_authority_request_timeout');
            }
            if (error instanceof TypeError) {
                throw new Error('sect_authority_request_network_error');
            }
            throw error;
        } finally {
            globalThis.clearTimeout(timeoutId);
        }
    }
}
