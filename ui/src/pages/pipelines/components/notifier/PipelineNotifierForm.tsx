import { useState } from "react";

import { TrashIcon } from "@phosphor-icons/react";

import Button, { ButtonVariant } from "@galaxy-io/dls/buttons/Button";
import FlexWrapper, {
  AlignItems,
  FlexDirection,
  FlexGap,
  JustifyContent,
} from "@galaxy-io/dls/containers/FlexWrapper";
import Widget from "@galaxy-io/dls/widget/Widget";

import PipelineNotifierFields from "@/pages/pipelines/components/notifier/PipelineNotifierFields";
import type { PipelineNotifierState } from "@/pages/pipelines/components/notifier/types";
import {
  hasPipelineNotifierChanges,
  isPipelineNotifierValid,
} from "@/pages/pipelines/components/notifier/utils";

interface PipelineNotifierFormProps {
  initialState: PipelineNotifierState;
  isSaving?: boolean;
  onSave: (state: PipelineNotifierState) => void;
  onCancel: () => void;
  onDelete?: () => void;
}

const PipelineNotifierForm = ({
  initialState,
  isSaving = false,
  onSave,
  onCancel,
  onDelete,
}: PipelineNotifierFormProps) => {
  const [state, setState] = useState<PipelineNotifierState>(initialState);

  const handleChange = (partial: Partial<PipelineNotifierState>) => {
    setState((prev) => ({ ...prev, ...partial }));
  };

  const handleSave = () => {
    onSave({ ...state, isEnabled: initialState.isEnabled });
  };

  const canSave = isPipelineNotifierValid(state) && hasPipelineNotifierChanges(state, initialState);

  return (
    <FlexWrapper padding="12px" fillWidth>
      <Widget noHover padding="16px" fillWidth>
        <FlexWrapper direction={FlexDirection.COLUMN} gap={FlexGap.MEDIUM} fillWidth>
          <PipelineNotifierFields state={state} onChange={handleChange} isDisabled={isSaving} />
          <FlexWrapper
            alignItems={AlignItems.CENTER}
            justifyContent={JustifyContent.END}
            gap={8}
            fillWidth
          >
            {onDelete && (
              <Button
                icon={TrashIcon}
                variant={ButtonVariant.SECONDARY}
                onClick={onDelete}
                isDisabled={isSaving}
                ariaLabel="Delete rule"
              />
            )}
            <Button
              label="Cancel"
              variant={ButtonVariant.SECONDARY}
              onClick={onCancel}
              isDisabled={isSaving}
            />
            <Button label="Save" onClick={handleSave} isDisabled={!canSave} isLoading={isSaving} />
          </FlexWrapper>
        </FlexWrapper>
      </Widget>
    </FlexWrapper>
  );
};

export default PipelineNotifierForm;
