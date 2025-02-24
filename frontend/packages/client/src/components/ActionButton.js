import React from 'react';
import Loader from 'components/Loader';
import classnames from 'classnames';

export default function ActionButton({
  enabled = true,
  onClick = () => {},
  loading = false,
  label = '',
  classNames,
  type,
} = {}) {
  const clNames = classnames(
    'button is-flex is-align-items-centered rounded-sm is-uppercase',
    'm-0 p-0',
    'has-background-yellow',
    { 'is-enabled': enabled },
    { 'is-disabled': !enabled },
    { [classNames]: !!classNames }
  );
  return (
    <button
      type={type}
      style={{ height: 48, width: '100%' }}
      className={clNames}
      onClick={!enabled ? () => {} : onClick}
    >
      {loading ? <Loader size={18} spacing="mx-button-loader" /> : <>{label}</>}
    </button>
  );
}
