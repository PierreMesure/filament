import type { Transport } from "@connectrpc/connect";
import {
  createConnectQueryKey,
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
} from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";

import type {
  ListPipelineNotifiersRequest,
  ListPipelineNotifiersResponse,
} from "@/gen/ingestion/v1/notifiers_pb";
import { IngestionService } from "@/gen/ingestion/v1/service_pb";

export const createListPipelineNotifiersQueryKey = (
  input?: ListPipelineNotifiersRequest,
  transport?: Transport,
) => {
  return createConnectQueryKey({
    schema: IngestionService.method.listPipelineNotifiers,
    input,
    transport,
    cardinality: undefined,
  });
};

export const useListPipelineNotifiersQuery = ({
  input,
  options = {},
}: {
  input: ListPipelineNotifiersRequest;
  options?: UseQueryOptions<
    typeof IngestionService.method.listPipelineNotifiers.output,
    ListPipelineNotifiersResponse
  >;
}) => {
  return useQuery<
    typeof IngestionService.method.listPipelineNotifiers.input,
    typeof IngestionService.method.listPipelineNotifiers.output
  >(IngestionService.method.listPipelineNotifiers, input, options);
};

export const useCreatePipelineNotifierMutation = (
  options: UseMutationOptions<
    typeof IngestionService.method.createPipelineNotifier.input,
    typeof IngestionService.method.createPipelineNotifier.output
  > = {},
) => {
  const queryClient = useQueryClient();
  return useMutation<
    typeof IngestionService.method.createPipelineNotifier.input,
    typeof IngestionService.method.createPipelineNotifier.output
  >(IngestionService.method.createPipelineNotifier, {
    ...options,
    onSettled: (...args) => {
      void queryClient.invalidateQueries({
        queryKey: createListPipelineNotifiersQueryKey(),
      });
      return options.onSettled?.(...args);
    },
  });
};

export const useUpdatePipelineNotifierMutation = (
  options: UseMutationOptions<
    typeof IngestionService.method.updatePipelineNotifier.input,
    typeof IngestionService.method.updatePipelineNotifier.output
  > = {},
) => {
  const queryClient = useQueryClient();
  return useMutation<
    typeof IngestionService.method.updatePipelineNotifier.input,
    typeof IngestionService.method.updatePipelineNotifier.output
  >(IngestionService.method.updatePipelineNotifier, {
    ...options,
    onSettled: (...args) => {
      void queryClient.invalidateQueries({
        queryKey: createListPipelineNotifiersQueryKey(),
      });
      return options.onSettled?.(...args);
    },
  });
};

export const useDeletePipelineNotifierMutation = (
  options: UseMutationOptions<
    typeof IngestionService.method.deletePipelineNotifier.input,
    typeof IngestionService.method.deletePipelineNotifier.output
  > = {},
) => {
  const queryClient = useQueryClient();
  return useMutation<
    typeof IngestionService.method.deletePipelineNotifier.input,
    typeof IngestionService.method.deletePipelineNotifier.output
  >(IngestionService.method.deletePipelineNotifier, {
    ...options,
    onSettled: (...args) => {
      void queryClient.invalidateQueries({
        queryKey: createListPipelineNotifiersQueryKey(),
      });
      return options.onSettled?.(...args);
    },
  });
};
