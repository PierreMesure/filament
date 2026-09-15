import type { PinnedOptions } from "@galaxy-io/dls/inputs/MultiSelectInput";
import type { SelectInputOption } from "@galaxy-io/dls/inputs/SelectInput";

import {
  PipelineNotifierEvent,
  type PipelineNotifierState,
} from "@/pages/pipelines/components/notifier/types";

export const PIPELINE_NOTIFIER_EVENT_ALL = "*";

export const PIPELINE_NOTIFIER_DESTINATION_SECRET_REF_KEY = "destination";

export const PIPELINE_NOTIFIER_INPUT_WIDTH = 351;
export const PIPELINE_NOTIFIER_TABLE_COLUMN_WIDTH_ENABLED = 64;
export const PIPELINE_NOTIFIER_TABLE_LOADING_ROW_COUNT = 3;

export const PIPELINE_NOTIFIER_URL_KEEP_PLACEHOLDER_TEXT = "Leave blank to keep current value";
export const PIPELINE_NOTIFIER_HEADERS_PLACEHOLDER_TEXT = `{
  "Authorization": "Bearer <token>"
}`;
export const PIPELINE_NOTIFIER_HEADERS_KEEP_PLACEHOLDER_TEXT = `{
  "Leave blank to keep current value": ""
}`;

export const PIPELINE_NOTIFIER_EVENT_OPTIONS: SelectInputOption[] = Object.values(
  PipelineNotifierEvent,
).map((event) => ({ id: event, label: event, value: event }));

export const PIPELINE_NOTIFIER_ALL_EVENTS_PINNED_OPTION: PinnedOptions = {
  id: "all-events",
  label: "All events",
  optionIds: PIPELINE_NOTIFIER_EVENT_OPTIONS.map((option) => option.id),
};

export const PIPELINE_NOTIFIER_DEFAULT_STATE: PipelineNotifierState = {
  name: "",
  isEnabled: false,
  events: [PipelineNotifierEvent.RUN_FAILED],
  hasStoredDestination: false,
  url: "",
  headers: "",
};
