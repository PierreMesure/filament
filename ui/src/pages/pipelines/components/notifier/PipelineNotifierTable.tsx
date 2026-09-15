import { useMemo, useState } from "react";

import { PlusIcon } from "@phosphor-icons/react";

import Button, { ButtonVariant } from "@galaxy-io/dls/buttons/Button";
import FlexItem from "@galaxy-io/dls/containers/FlexItem";
import FlexWrapper, { AlignItems, FlexDirection } from "@galaxy-io/dls/containers/FlexWrapper";
import HorizontalDivider from "@galaxy-io/dls/dividers/HorizontalDivider";
import { InputSize } from "@galaxy-io/dls/inputs/Input";
import TextInput from "@galaxy-io/dls/inputs/TextInput";
import ToggleInput from "@galaxy-io/dls/inputs/ToggleInput";
import InfiniteTable, {
  ColumnAlign,
  type ColumnDef,
  TableVariant,
} from "@galaxy-io/dls/table/InfiniteTable";
import Text, { TextSize } from "@galaxy-io/dls/text/Text";
import TextShimmer from "@galaxy-io/dls/text/TextShimmer";
import Widget from "@galaxy-io/dls/widget/Widget";

import EmptyLayout, { EmptyLayoutSize } from "@/layouts/EmptyLayout";

import {
  PIPELINE_NOTIFIER_DEFAULT_STATE,
  PIPELINE_NOTIFIER_TABLE_COLUMN_WIDTH_ENABLED,
  PIPELINE_NOTIFIER_TABLE_LOADING_ROW_COUNT,
} from "@/pages/pipelines/components/notifier/constants";
import PipelineNotifierForm from "@/pages/pipelines/components/notifier/PipelineNotifierForm";
import type {
  PipelineNotifierSaveCallbacks,
  PipelineNotifierState,
  PipelineNotifierTableRow,
} from "@/pages/pipelines/components/notifier/types";

import { isSearchMatch } from "@/utils/search";

const CREATE_STATE: PipelineNotifierState = {
  ...PIPELINE_NOTIFIER_DEFAULT_STATE,
  isEnabled: true,
};

interface PipelineNotifierTableProps {
  rows: PipelineNotifierTableRow[];
  isLoading?: boolean;
  isSaving?: boolean;
  onCreate: (state: PipelineNotifierState, callbacks: PipelineNotifierSaveCallbacks) => void;
  onUpdate: (
    id: PipelineNotifierTableRow["id"],
    state: PipelineNotifierState,
    callbacks: PipelineNotifierSaveCallbacks,
  ) => void;
  onDelete: (id: PipelineNotifierTableRow["id"]) => void;
  onToggleEnabled: (id: PipelineNotifierTableRow["id"], isEnabled: boolean) => void;
}

interface PipelineNotifierTableState {
  search: string;
  isCreating: boolean;
  expandedRowIds: PipelineNotifierTableRow["id"][];
}

const DEFAULT_STATE: PipelineNotifierTableState = {
  search: "",
  isCreating: false,
  expandedRowIds: [],
};

const PipelineNotifierTable = ({
  rows,
  isLoading = false,
  isSaving = false,
  onCreate,
  onUpdate,
  onDelete,
  onToggleEnabled,
}: PipelineNotifierTableProps) => {
  const [state, setState] = useState<PipelineNotifierTableState>(DEFAULT_STATE);

  const filteredRows = rows.filter((row) => isSearchMatch(state.search, row.state.name));

  const handleSearchChange = (search: string) => {
    setState((prev) => ({ ...prev, search }));
  };

  const handleCreatingChange = (isCreating: boolean) => {
    setState((prev) => ({ ...prev, isCreating }));
  };

  const handleExpandedChange = (expandedRowIds: PipelineNotifierTableRow["id"][]) => {
    setState((prev) => ({ ...prev, expandedRowIds }));
  };

  const columns = useMemo<ColumnDef<PipelineNotifierTableRow>[]>(
    () => [
      {
        id: "name",
        header: "Name",
        accessorFn: (row) => row.state.name,
        enableSorting: false,
        cellLoading: () => <TextShimmer width={140} height={16} />,
        cell: ({ row }) => (
          <Text size={TextSize.BODY_SM} isEllipsis>
            {row.original.state.name}
          </Text>
        ),
      },
      {
        id: "enabled",
        header: "",
        align: ColumnAlign.RIGHT,
        size: PIPELINE_NOTIFIER_TABLE_COLUMN_WIDTH_ENABLED,
        enableSorting: false,
        cellLoading: () => null,
        cell: ({ row }) => (
          <ToggleInput
            size={InputSize.LARGE}
            value={row.original.state.isEnabled}
            onChange={(isEnabled) => onToggleEnabled(row.original.id, isEnabled)}
            isDisabled={isSaving}
          />
        ),
      },
    ],
    [isSaving, onToggleEnabled],
  );

  return (
    <Widget noPadding noHover fillWidth>
      <FlexWrapper direction={FlexDirection.COLUMN} fillWidth>
        <FlexWrapper alignItems={AlignItems.CENTER} gap={8} padding="8px" fillWidth>
          <TextInput
            placeholder="Search"
            value={state.search}
            onChange={handleSearchChange}
            fillWidth
          />
          <FlexItem shrink={0}>
            <Button
              label="Add notifier"
              icon={PlusIcon}
              variant={ButtonVariant.SECONDARY}
              onClick={() => handleCreatingChange(true)}
              isDisabled={state.isCreating || isSaving}
            />
          </FlexItem>
        </FlexWrapper>
        <HorizontalDivider />
        {state.isCreating && (
          <>
            <PipelineNotifierForm
              initialState={CREATE_STATE}
              isSaving={isSaving}
              onSave={(next) => onCreate(next, { onSuccess: () => handleCreatingChange(false) })}
              onCancel={() => handleCreatingChange(false)}
            />
            <HorizontalDivider />
          </>
        )}
        <InfiniteTable<PipelineNotifierTableRow>
          columns={columns}
          data={filteredRows}
          getRowId={(row) => row.id}
          isLoading={isLoading}
          loadingRowCount={PIPELINE_NOTIFIER_TABLE_LOADING_ROW_COUNT}
          contentWhenEmpty={
            <EmptyLayout
              size={EmptyLayoutSize.SMALL}
              header={rows.length === 0 ? "No notifiers" : undefined}
              message={
                rows.length === 0
                  ? "Add a notifier to get notified when runs complete or fail."
                  : "No rules match your search."
              }
            />
          }
          expandedRowIds={state.expandedRowIds}
          onExpandedChange={handleExpandedChange}
          onRowExpand={(row) => (
            <PipelineNotifierForm
              initialState={row.original.state}
              isSaving={isSaving}
              onSave={(next) =>
                onUpdate(row.original.id, next, { onSuccess: () => handleExpandedChange([]) })
              }
              onCancel={() => handleExpandedChange([])}
              onDelete={() => onDelete(row.original.id)}
            />
          )}
          enableMultiRowExpansion={false}
          variant={TableVariant.PRIMARY}
          fillWidth
        />
      </FlexWrapper>
    </Widget>
  );
};

export default PipelineNotifierTable;
