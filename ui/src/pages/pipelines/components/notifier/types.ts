import type { Notifier, WebhookNotifierDestination } from "@/gen/ingestion/v1/notifiers_pb";

// TODO(@mitchbregs): This should be a proto
export enum PipelineNotifierEvent {
  RUN_COMPLETED = "run.completed",
  RUN_FAILED = "run.failed",
}

export interface PipelineNotifierState {
  name: Notifier["name"];
  isEnabled: Notifier["isEnabled"];
  events: PipelineNotifierEvent[];
  hasStoredDestination: boolean;
  url: WebhookNotifierDestination["url"];
  headers: string;
}

export interface PipelineNotifier extends PipelineNotifierState {
  id: Notifier["id"];
}
