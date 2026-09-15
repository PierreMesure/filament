import { create } from "@bufbuild/protobuf";
import pluralize from "pluralize";

import type { SelectInputOption } from "@galaxy-io/dls/inputs/SelectInput";

import {
  NotificationType,
  type Notifier,
  type NotifierInput,
  NotifierInputSchema,
  type WebhookNotifierDestination,
  WebhookNotifierDestinationSchema,
  WebhookNotifierInputSchema,
} from "@/gen/ingestion/v1/notifiers_pb";

import { isNameValid } from "@/pages/connectors/components/form/validation";
import {
  PIPELINE_NOTIFIER_ALL_EVENTS_PINNED_OPTION,
  PIPELINE_NOTIFIER_DESTINATION_SECRET_REF_KEY,
  PIPELINE_NOTIFIER_EVENT_ALL,
  PIPELINE_NOTIFIER_EVENTS,
} from "@/pages/pipelines/components/notifier/constants";
import type {
  PipelineNotifierEvent,
  PipelineNotifierState,
} from "@/pages/pipelines/components/notifier/types";

export const parsePipelineNotifierEvents = (events: string[]): PipelineNotifierEvent[] => {
  if (events.includes(PIPELINE_NOTIFIER_EVENT_ALL)) return [...PIPELINE_NOTIFIER_EVENTS];
  return PIPELINE_NOTIFIER_EVENTS.filter((event) => events.includes(event));
};

export const formatPipelineNotifierEventsSelection = (
  selectedOptions: SelectInputOption[],
  placeholder: string,
): string => {
  if (selectedOptions.length === 0) return placeholder;
  if (selectedOptions.length === 1) return selectedOptions[0].label;
  if (selectedOptions.length === PIPELINE_NOTIFIER_EVENTS.length) {
    return PIPELINE_NOTIFIER_ALL_EVENTS_PINNED_OPTION.label;
  }
  return pluralize("event", selectedOptions.length, true);
};

export const isPipelineNotifierUrlValid = (url: string): boolean => {
  try {
    const { protocol } = new URL(url.trim());
    return protocol === "http:" || protocol === "https:";
  } catch {
    return false;
  }
};

export const parsePipelineNotifierHeaders = (
  text: string,
): WebhookNotifierDestination["headers"] | null => {
  if (text.trim() === "") return {};
  try {
    const parsed: unknown = JSON.parse(text);
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) return null;
    const headers = parsed as Record<string, unknown>;
    return Object.values(headers).every((value) => typeof value === "string")
      ? (headers as WebhookNotifierDestination["headers"])
      : null;
  } catch {
    return null;
  }
};

const isKeepingStoredDestination = (state: PipelineNotifierState): boolean =>
  state.hasStoredDestination && state.url.trim() === "" && state.headers.trim() === "";

export const isPipelineNotifierValid = (state: PipelineNotifierState): boolean =>
  isNameValid(state.name) &&
  state.events.length > 0 &&
  (isKeepingStoredDestination(state) ||
    (isPipelineNotifierUrlValid(state.url) &&
      parsePipelineNotifierHeaders(state.headers) !== null));

export const mapPipelineNotifierStateToInput = (state: PipelineNotifierState): NotifierInput =>
  create(NotifierInputSchema, {
    name: state.name.trim(),
    notificationType: NotificationType.WEBHOOK,
    isEnabled: state.isEnabled,
    events: parsePipelineNotifierEvents(state.events),
    channel: {
      case: "webhook",
      value: create(
        WebhookNotifierInputSchema,
        isKeepingStoredDestination(state)
          ? {}
          : {
              destinationSource: {
                case: "destination",
                value: create(WebhookNotifierDestinationSchema, {
                  url: state.url.trim(),
                  headers: parsePipelineNotifierHeaders(state.headers) ?? {},
                }),
              },
            },
      ),
    },
  });

export const mapNotifierToPipelineNotifierState = (notifier: Notifier): PipelineNotifierState => ({
  name: notifier.name,
  isEnabled: notifier.isEnabled,
  events: parsePipelineNotifierEvents(notifier.events),
  hasStoredDestination: PIPELINE_NOTIFIER_DESTINATION_SECRET_REF_KEY in notifier.secretRefs,
  url: "",
  headers: "",
});
