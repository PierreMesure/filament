import type { Notifier, NotifierEvent } from "@/gen/ingestion/v1/notifiers_pb";

export interface PipelineNotifierState {
  name: Notifier["name"];
  isEnabled: Notifier["isEnabled"];
  events: NotifierEvent[];
  url: string;
  headers: string;
  secretRefs: Notifier["secretRefs"];
}

export interface PipelineNotifier extends PipelineNotifierState {
  id: Notifier["id"];
}
