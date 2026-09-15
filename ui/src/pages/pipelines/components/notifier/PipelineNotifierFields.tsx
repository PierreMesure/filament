import FlexWrapper, {
  AlignItems,
  FlexDirection,
  FlexGap,
  JustifyContent,
} from "@galaxy-io/dls/containers/FlexWrapper";
import CodeEditor from "@galaxy-io/dls/editor/CodeEditor";
import { InputSize } from "@galaxy-io/dls/inputs/Input";
import MultiSelectInput from "@galaxy-io/dls/inputs/MultiSelectInput";
import type { SelectInputOption } from "@galaxy-io/dls/inputs/SelectInput";
import TextInput from "@galaxy-io/dls/inputs/TextInput";
import Text, { TextSize, TextVariant } from "@galaxy-io/dls/text/Text";

import {
  PIPELINE_NOTIFIER_EVENT_OPTIONS,
  PIPELINE_NOTIFIER_HEADERS_PLACEHOLDER_TEXT,
  PIPELINE_NOTIFIER_INPUT_WIDTH,
  PIPELINE_NOTIFIER_KEEP_PLACEHOLDER_TEXT,
} from "@/pages/pipelines/components/notifier/constants";
import type {
  PipelineNotifierEvent,
  PipelineNotifierState,
} from "@/pages/pipelines/components/notifier/types";
import {
  formatPipelineNotifierEventsSelection,
  isPipelineNotifierUrlValid,
  parsePipelineNotifierHeaders,
} from "@/pages/pipelines/components/notifier/utils";

interface PipelineNotifierFieldsProps {
  state: PipelineNotifierState;
  onChange: (partial: Partial<PipelineNotifierState>) => void;
  isDisabled?: boolean;
}

const PipelineNotifierFields = ({
  state,
  onChange,
  isDisabled = false,
}: PipelineNotifierFieldsProps) => {
  const urlError =
    state.url !== "" && !isPipelineNotifierUrlValid(state.url)
      ? "Use an absolute http or https URL"
      : undefined;
  const headersError =
    parsePipelineNotifierHeaders(state.headers) === null
      ? "Use a JSON object with string values"
      : undefined;
  const urlPlaceholder = state.hasStoredDestination
    ? PIPELINE_NOTIFIER_KEEP_PLACEHOLDER_TEXT
    : "https://example.com/hooks/filament";
  const headersPlaceholder = state.hasStoredDestination
    ? PIPELINE_NOTIFIER_KEEP_PLACEHOLDER_TEXT
    : PIPELINE_NOTIFIER_HEADERS_PLACEHOLDER_TEXT;
  const selectedEventOptions = PIPELINE_NOTIFIER_EVENT_OPTIONS.filter((option) =>
    state.events.includes(option.value as PipelineNotifierEvent),
  );

  const handleNameChange = (name: string) => {
    onChange({ name });
  };

  const handleEventsChange = (options: SelectInputOption[]) => {
    onChange({ events: options.map((option) => option.value as PipelineNotifierEvent) });
  };

  const handleUrlChange = (url: string) => {
    onChange({ url });
  };

  const handleHeadersChange = (headers: string) => {
    onChange({ headers });
  };

  return (
    <FlexWrapper direction={FlexDirection.COLUMN} gap={FlexGap.MEDIUM} fillWidth>
      <FlexWrapper
        alignItems={AlignItems.CENTER}
        justifyContent={JustifyContent.SPACE_BETWEEN}
        fillWidth
      >
        <Text variant={TextVariant.SECONDARY}>Name</Text>
        <TextInput
          value={state.name}
          onChange={handleNameChange}
          placeholder="Notify on-call"
          size={InputSize.LARGE}
          width={PIPELINE_NOTIFIER_INPUT_WIDTH}
          isDisabled={isDisabled}
        />
      </FlexWrapper>
      <FlexWrapper
        alignItems={AlignItems.CENTER}
        justifyContent={JustifyContent.SPACE_BETWEEN}
        fillWidth
      >
        <Text variant={TextVariant.SECONDARY}>Events</Text>
        <MultiSelectInput
          options={PIPELINE_NOTIFIER_EVENT_OPTIONS}
          value={selectedEventOptions}
          onChange={handleEventsChange}
          renderSelectedText={formatPipelineNotifierEventsSelection}
          placeholder="Select events"
          size={InputSize.LARGE}
          width={PIPELINE_NOTIFIER_INPUT_WIDTH}
          isDisabled={isDisabled}
        />
      </FlexWrapper>
      <FlexWrapper
        alignItems={AlignItems.CENTER}
        justifyContent={JustifyContent.SPACE_BETWEEN}
        fillWidth
      >
        <Text variant={TextVariant.SECONDARY}>URL</Text>
        <TextInput
          value={state.url}
          onChange={handleUrlChange}
          error={urlError}
          placeholder={urlPlaceholder}
          size={InputSize.LARGE}
          width={PIPELINE_NOTIFIER_INPUT_WIDTH}
          isDisabled={isDisabled}
        />
      </FlexWrapper>
      <FlexWrapper
        alignItems={AlignItems.START}
        justifyContent={JustifyContent.SPACE_BETWEEN}
        fillWidth
      >
        <Text variant={TextVariant.SECONDARY}>Headers</Text>
        <FlexWrapper direction={FlexDirection.COLUMN} gap={4} width={PIPELINE_NOTIFIER_INPUT_WIDTH}>
          <CodeEditor
            content={state.headers}
            onChange={handleHeadersChange}
            lang="json"
            placeholder={headersPlaceholder}
            borderRadius={4}
            isReadOnly={isDisabled}
            noLineNumbers
          />
          {headersError && (
            <Text size={TextSize.BODY_SM} variant={TextVariant.ERROR}>
              {headersError}
            </Text>
          )}
        </FlexWrapper>
      </FlexWrapper>
    </FlexWrapper>
  );
};

export default PipelineNotifierFields;
