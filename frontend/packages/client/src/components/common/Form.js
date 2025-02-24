import React from 'react';

const Form = ({ removeInnerForm = false, children, onSubmit, formId } = {}) => {
  // TODO: make enter to jump to next input field on form
  const checkKeyDown = (e) => {
    if (e.code === 'Enter') e.preventDefault();
  };
  return removeInnerForm ? (
    <>{children}</>
  ) : (
    <form onSubmit={onSubmit} id={formId} onKeyDown={(e) => checkKeyDown(e)}>
      {children}
    </form>
  );
};

export default Form;
