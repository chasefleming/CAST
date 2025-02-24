import { useErrorHandlerContext } from '../contexts/ErrorHandler';
import { useWebContext } from 'contexts/Web3';
import { UPDATE_MEMBERSHIP_TX } from 'const';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  addUserToCommunityUserApiRep,
  deleteCommunityMemberApiReq,
} from 'api/communityUsers';

export default function useJoinCommunity() {
  const queryClient = useQueryClient();
  const { notifyError } = useErrorHandlerContext();
  const { signMessageByWalletProvider } = useWebContext();

  const { mutate: createCommunityUserMutation } = useMutation(
    async ({ communityId, user }) => {
      const { addr } = user;
      const hexTime = Buffer.from(Date.now().toString()).toString('hex');
      const [compositeSignatures, voucher] = await signMessageByWalletProvider(
        user?.services[0].uid,
        UPDATE_MEMBERSHIP_TX,
        hexTime
      );

      if (!compositeSignatures && !voucher) {
        throw new Error('No valid user signature found.');
      }

      return addUserToCommunityUserApiRep({
        communityId,
        addr,
        hexTime,
        compositeSignatures,
        voucher,
        userType: 'member',
        signingAddr: addr,
      });
    },
    {
      onSuccess: async (_, variables) => {
        const {
          user: { addr },
        } = variables;

        await queryClient.invalidateQueries('communities-for-homepage');
        await queryClient.invalidateQueries([
          'connected-user-communities',
          addr,
        ]);
      },
      onError: (error) => {
        notifyError(error);
      },
    }
  );

  const { mutate: deleteUserFromCommunityMutation } = useMutation(
    async ({ communityId, user }) => {
      const { addr } = user;
      const hexTime = Buffer.from(Date.now().toString()).toString('hex');

      const [compositeSignatures, voucher] = await signMessageByWalletProvider(
        user?.services[0].uid,
        UPDATE_MEMBERSHIP_TX,
        hexTime
      );

      if (!compositeSignatures && !voucher) {
        throw new Error('No valid user signature found.');
      }

      return deleteCommunityMemberApiReq({
        communityId,
        addr,
        hexTime,
        compositeSignatures,
        voucher,
        signingAddr: addr,
      });
    },
    {
      onSuccess: async (_, variables) => {
        const {
          user: { addr },
        } = variables;

        await queryClient.invalidateQueries('communities-for-homepage');
        await queryClient.invalidateQueries([
          'connected-user-communities',
          addr,
        ]);
      },
      onError: (error) => {
        notifyError(error);
      },
    }
  );

  return {
    // adds user as member
    createCommunityUser: createCommunityUserMutation,
    // removes all roles from user
    deleteUserFromCommunity: deleteUserFromCommunityMutation,
  };
}
