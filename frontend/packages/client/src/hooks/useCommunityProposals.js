import { useErrorHandlerContext } from 'contexts/ErrorHandler';
import { getPagination, getPlainData } from 'utils';
import { useInfiniteQuery } from '@tanstack/react-query';
import { fetchProposalsByStatus } from 'api/proposals';
import { PAGINATION_INITIAL_STATE } from '../reducers';

/**
 * Hook to return proposals created on a community. Results are paginated
 * @param  {int} communityId community Id to get proposals from
 * @param  {int} count page size, used for pagination limiting the number of elements returned. Defaults to 10. Max value is 25.
 * @param  {int} start indicates the start index for paginated results. Defaults to 0.
 */
export default function useCommunityProposals({
  communityId,
  start: startParam = PAGINATION_INITIAL_STATE.start,
  count: countParam = PAGINATION_INITIAL_STATE.count,
  status,
} = {}) {
  const initialPageParam = [startParam, countParam, 0, -1];

  const { notifyError } = useErrorHandlerContext();

  const queryUniqueKey = ['community-proposals', String(communityId), status];

  const { isLoading, isError, data, error, fetchNextPage } = useInfiniteQuery(
    queryUniqueKey,
    async ({ pageParam = initialPageParam, queryKey }) => {
      const [start, count] = pageParam;
      const communityId = queryKey[1];
      return fetchProposalsByStatus({ communityId, count, start, status });
    },
    {
      getNextPageParam: (lastPage, pages) => {
        const { next, start, count, totalRecords } = lastPage;
        return [start + count, count, totalRecords, next];
      },
      enabled: !!communityId,
      onError: (error) => {
        notifyError(error);
      },
      keepPreviousData: true,
    }
  );

  return {
    isLoading,
    isError,
    data: getPlainData(data),
    pagination: getPagination(data, countParam),
    error,
    fetchNextPage,
    queryKey: queryUniqueKey,
    pages: data?.pages ?? [],
  };
}
