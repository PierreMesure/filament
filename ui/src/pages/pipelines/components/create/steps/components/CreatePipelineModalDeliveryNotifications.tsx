import { CreatePipelineModalActionType } from "@/pages/pipelines/components/create/actions";
import {
  useCreatePipelineModalDispatch,
  useCreatePipelineModalState,
} from "@/pages/pipelines/components/create/CreatePipelineModalProvider";
import type { CreatePipelineModalNotifier } from "@/pages/pipelines/components/create/types";
import PipelineNotifierTable from "@/pages/pipelines/components/notifier/PipelineNotifierTable";
import type {
  PipelineNotifierSaveCallbacks,
  PipelineNotifierState,
} from "@/pages/pipelines/components/notifier/types";

const CreatePipelineModalDeliveryNotifications = () => {
  const { notifiers } = useCreatePipelineModalState();
  const dispatch = useCreatePipelineModalDispatch();

  const rows = notifiers.map(({ id, ...state }) => ({ id, state }));

  const handleCreate = (
    state: PipelineNotifierState,
    { onSuccess }: PipelineNotifierSaveCallbacks,
  ) => {
    dispatch({
      type: CreatePipelineModalActionType.ADD_NOTIFIER,
      payload: { ...state, id: crypto.randomUUID() },
    });
    onSuccess();
  };

  const handleUpdate = (
    id: CreatePipelineModalNotifier["id"],
    state: PipelineNotifierState,
    { onSuccess }: PipelineNotifierSaveCallbacks,
  ) => {
    dispatch({
      type: CreatePipelineModalActionType.UPDATE_NOTIFIER,
      payload: { id, partial: state },
    });
    onSuccess();
  };

  const handleToggleEnabled = (id: CreatePipelineModalNotifier["id"], isEnabled: boolean) => {
    dispatch({
      type: CreatePipelineModalActionType.UPDATE_NOTIFIER,
      payload: { id, partial: { isEnabled } },
    });
  };

  const handleDelete = (id: CreatePipelineModalNotifier["id"]) => {
    dispatch({ type: CreatePipelineModalActionType.REMOVE_NOTIFIER, payload: id });
  };

  return (
    <PipelineNotifierTable
      rows={rows}
      onCreate={handleCreate}
      onUpdate={handleUpdate}
      onDelete={handleDelete}
      onToggleEnabled={handleToggleEnabled}
    />
  );
};

export default CreatePipelineModalDeliveryNotifications;
