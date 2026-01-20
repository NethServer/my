//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

// import { getUserRoles, USER_ROLES_KEY } from '@/lib/userRoles' ////
// import { useLoginStore } from '@/stores/login'
// import { defineQuery, useQuery } from '@pinia/colada'

//// is this used?

////
// export const useUserRoles = defineQuery(() => {
//   const loginStore = useLoginStore()

//   const { state, asyncStatus, ...rest } = useQuery({
//     key: () => [USER_ROLES_KEY],
//     enabled: () => !!loginStore.jwtToken,
//     query: () => getUserRoles(),
//   })

//   return {
//     ...rest,
//     state,
//     asyncStatus,
//   }
// })
