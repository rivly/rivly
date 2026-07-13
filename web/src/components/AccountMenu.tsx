import { Menu } from '@base-ui/react/menu'
import { useNavigate } from '@tanstack/react-router'
import { LuChevronDown, LuLogOut } from 'react-icons/lu'
import { useLogout, useMe } from '../lib/auth'
import styles from './AccountMenu.module.css'

export function AccountMenu() {
  const navigate = useNavigate()
  const { data: me } = useMe()
  const logout = useLogout()

  const name = me?.displayName || me?.email || ''
  const initial = name.charAt(0).toUpperCase()

  return (
    <Menu.Root>
      <Menu.Trigger className={styles.trigger}>
        <span className={styles.avatar}>{initial}</span>
        <span className={styles.name}>{name}</span>
        <LuChevronDown className={styles.caret} />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner className={styles.positioner} sideOffset={8} align="end">
          <Menu.Popup className={styles.popup}>
            <div className={styles.info}>
              {me?.displayName && (
                <span className={styles.infoName}>{me.displayName}</span>
              )}
              <span className={styles.infoEmail}>{me?.email}</span>
            </div>
            <Menu.Separator className={styles.separator} />
            <Menu.Item
              className={styles.item}
              onClick={() =>
                logout.mutate(undefined, {
                  onSuccess: () => navigate({ to: '/login' }),
                })
              }
            >
              <LuLogOut className={styles.itemIcon} />
              Log out
            </Menu.Item>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  )
}
