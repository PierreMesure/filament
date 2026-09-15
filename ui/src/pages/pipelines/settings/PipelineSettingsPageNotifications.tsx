import { useMemo } from "react";

import { create } from "@bufbuild/protobuf";
import { useParams } from "@tanstack/react-router";

import Accordion from "@galaxy-io/dls/accordion/Accordion";
import { ButtonVariant } from "@galaxy-io/dls/buttons/Button";
import { ToastVariant } from "@galaxy-io/dls/toast/Toast";
import { useToast } from "@galaxy-io/dls/toast/useToast";

import {
  CreatePipelineNotifierRequestSchema,
  DeletePipelineNotifierRequestSchema,
  ListPipelineNotifiersRequestSchema,
  type Notifier,
  UpdatePipelineNotifierRequestSchema,
} from "@/gen/ingestion/v1/notifiers_pb";

import Dialog from "@/components/Dialog";

import PipelineNotifierTable from "@/pages/pipelines/components/notifier/PipelineNotifierTable";
import type {
  PipelineNotifierSaveCallbacks,
  PipelineNotifierState,
} from "@/pages/pipelines/components/notifier/types";
import {
  mapNotifierToPipelineNotifierState,
  mapNotifierToPipelineNotifierTableRow,
  mapPipelineNotifierStateToInput,
} from "@/pages/pipelines/components/notifier/utils";

import {
  useCreatePipelineNotifierMutation,
  useDeletePipelineNotifierMutation,
  useListPipelineNotifiersQuery,
  useUpdatePipelineNotifierMutation,
} from "@/api/queries/notifiers";

import { useConfirm } from "@/hooks/useConfirm";

import { getErrorMessage } from "@/utils/errors";

const PipelineSettingsPageNotifications = () => {
  const { showToast } = useToast();
  const { id: pipelineId } = useParams({ from: "/_app/pipelines/$id" });

  const { data, isLoading } = useListPipelineNotifiersQuery({
    input: create(ListPipelineNotifiersRequestSchema, { pipelineId }),
  });
  const notifiers = useMemo(() => data?.notifiers ?? [], [data]);
  const rows = useMemo(() => notifiers.map(mapNotifierToPipelineNotifierTableRow), [notifiers]);

  const { mutate: createNotifier, isPending: isCreating } = useCreatePipelineNotifierMutation();
  const { mutate: updateNotifier, isPending: isUpdating } = useUpdatePipelineNotifierMutation();
  const { mutate: deleteNotifier, isPending: isDeleting } = useDeletePipelineNotifierMutation();

  const { handleOpen, isOpen, target, handleClose, handleConfirm } = useConfirm<Notifier>({
    entityLabel: "Notification rule",
    entityName: (notifier) => notifier.name,
    onConfirm: (notifier, { onSuccess, onError }) =>
      deleteNotifier(
        create(DeletePipelineNotifierRequestSchema, {
          pipelineId,
          notifierId: notifier.id,
          version: notifier.version,
        }),
        { onSuccess, onError },
      ),
  });

  const handleCreate = (
    state: PipelineNotifierState,
    { onSuccess }: PipelineNotifierSaveCallbacks,
  ) => {
    createNotifier(
      create(CreatePipelineNotifierRequestSchema, {
        pipelineId,
        notifier: mapPipelineNotifierStateToInput(state),
      }),
      {
        onSuccess: () => {
          showToast({
            header: "Notification rule created",
            subheader: "Your pipeline will now send notifications.",
            variant: ToastVariant.SUCCESS,
          });
          onSuccess();
        },
        onError: (error) => {
          showToast({
            header: "Create failed",
            subheader: getErrorMessage(error, "Failed to create notification rule"),
            variant: ToastVariant.ERROR,
          });
        },
      },
    );
  };

  const handleUpdate = (
    id: Notifier["id"],
    state: PipelineNotifierState,
    { onSuccess }: PipelineNotifierSaveCallbacks,
  ) => {
    const notifier = notifiers.find((candidate) => candidate.id === id);
    if (!notifier) return;

    updateNotifier(
      create(UpdatePipelineNotifierRequestSchema, {
        pipelineId,
        notifierId: notifier.id,
        version: notifier.version,
        notifier: mapPipelineNotifierStateToInput(state),
      }),
      {
        onSuccess: () => {
          showToast({
            header: "Notification rule saved",
            subheader: "Your notification rule has been saved successfully.",
            variant: ToastVariant.SUCCESS,
          });
          onSuccess();
        },
        onError: (error) => {
          showToast({
            header: "Save failed",
            subheader: getErrorMessage(error, "Failed to save notification rule"),
            variant: ToastVariant.ERROR,
          });
        },
      },
    );
  };

  const handleToggleEnabled = (id: Notifier["id"], isEnabled: boolean) => {
    const notifier = notifiers.find((candidate) => candidate.id === id);
    if (!notifier) return;

    updateNotifier(
      create(UpdatePipelineNotifierRequestSchema, {
        pipelineId,
        notifierId: notifier.id,
        version: notifier.version,
        notifier: mapPipelineNotifierStateToInput({
          ...mapNotifierToPipelineNotifierState(notifier),
          isEnabled,
        }),
      }),
      {
        onSuccess: () => {
          showToast({
            header: isEnabled ? "Notification rule enabled" : "Notification rule disabled",
            subheader: isEnabled
              ? `${notifier.name} will now send notifications.`
              : `${notifier.name} will no longer send notifications.`,
            variant: ToastVariant.SUCCESS,
          });
        },
        onError: (error) => {
          showToast({
            header: "Update failed",
            subheader: getErrorMessage(error, "Failed to update notification rule"),
            variant: ToastVariant.ERROR,
          });
        },
      },
    );
  };

  const handleDelete = (id: Notifier["id"]) => {
    const notifier = notifiers.find((candidate) => candidate.id === id);
    if (notifier) handleOpen(notifier);
  };

  return (
    <Accordion header="Notifications" isOpenInitial>
      <PipelineNotifierTable
        rows={rows}
        isLoading={isLoading}
        isSaving={isCreating || isUpdating || isDeleting}
        onCreate={handleCreate}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
        onToggleEnabled={handleToggleEnabled}
      />
      <Dialog
        open={isOpen}
        onClose={handleClose}
        onConfirm={handleConfirm}
        title="Delete notification rule"
        body="This is a destructive action and cannot be undone."
        confirmationPhrase={target?.name}
        confirmLabel="Delete rule"
        confirmVariant={ButtonVariant.ERROR}
        isPending={isDeleting}
      />
    </Accordion>
  );
};

export default PipelineSettingsPageNotifications;
