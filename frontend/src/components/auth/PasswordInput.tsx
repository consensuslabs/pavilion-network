import React, { FC, useState } from 'react';
import { Box, TextInput } from 'grommet';
import { Hide, View } from 'grommet-icons';

interface PasswordInputProps {
  id?: string;
  name: string;
}

const PasswordInput: FC<PasswordInputProps> = ({ id, name }) => {
  const [revealed, setRevealed] = useState(false);
  const onToggleReveal = () => {
    setRevealed(!revealed);
  };
  return (
    <Box
      direction="row"
      justify="between"
      align="center"
      gap="small"
      round="small"
    >
      <TextInput
        id={id}
        name={name}
        type={revealed ? 'text' : 'password'}
        plain
        focusIndicator={false}
      />
      <Box
        focusIndicator={false}
        pad={{ right: 'small' }}
        onClick={onToggleReveal}
      >
        {revealed ? <View size="medium" /> : <Hide size="medium" />}
      </Box>
    </Box>
  );
};

export default PasswordInput;
