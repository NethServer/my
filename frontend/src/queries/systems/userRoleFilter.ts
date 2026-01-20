//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

//// is this used?

////
// export const useUserRoleFilter = defineQuery(() => {
//   const loginStore = useLoginStore()

//   const { state, asyncStatus, ...rest } = useQuery({
//     key: () => [USER_ROLE_FILTER_KEY],
//     enabled: () => !!loginStore.jwtToken,
//     query: () => getUserRoleFilter(),
//   })

//   return {
//     ...rest,
//     state,
//     asyncStatus,
//   }
// })
