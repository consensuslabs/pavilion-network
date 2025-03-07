import { base, grommet, ThemeType } from 'grommet';
import { deepMerge } from 'grommet/utils';
import { CaretDownFill, CaretRightFill } from 'grommet-icons';

export const theme: typeof grommet = deepMerge(base, {
  global: {
    font: {
      family: 'Stara, DM Sans, sans-serif',
    },
    colors: {
      brand: '#9060EB',
      'brand-alt': '#7D4CDB',
      'brand-dark': '#643FBB',
      'status-disabled': '#8F9BB3',
      'status-disabled-light': '#E4E9F2',
      'status-unknown-light': '#E4E9F2',
      'status-info': '#16A8DE',
      blue: '#0091FF',
      dark: '#202631',
      'baby-pink': '#FFEEEB',
      'soft-pink': '#FFE7E3',
      'grayish-blue': '#3B5998',
      'light-turquoise': '#74C0BF',
      aqua: '#6FFFB0',
    },
    input: {
      extend: 'color: grey;',
      padding: { vertical: '12px' },
      font: {
        weight: 'normal',
        size: 'small',
      },
    },
    elevation: {
      light: {
        indigo: '-5px 5px 30px #3D138D26',
        indigoLight: '0px 0px 10px #3D138D26',
        peach: '0px 2px 20px 0px #FFE7E380',
        left: '-2px 0px 3px rgb(255 231 227 / 50%)',
        right: '2px 0px 3px rgb(255 231 227 / 50%)',
      },
      dark: {
        indigo: '0 5px 15px #3D138D35',
        left: '-2px 0px 3px rgb(255 231 227 / 50%)',
        right: '2px 0px 3px rgb(255 231 227 / 50%)',
      },
    },
    control: {
      border: {
        color: 'light-4',
        radius: '50px',
      },
    },
  },
  checkBox: {
    hover: {
      border: {
        color: 'accent-1',
      },
    },
  },
  layer: {
    border: {
      radius: '20px',
      intelligentRounding: true,
    },
  },
  button: {
    primary: {
      extend: 'color: white;',
    },
  },
  formField: {
    border: {
      side: 'all',
      color: 'brand',
    },
    round: 'small',
    label: {
      margin: { horizontal: 'small' },
      size: 'small',
      color: {
        light: '#202631',
      },
    },
    error: {
      background: {
        color: 'status-critical',
        opacity: 'weak',
      },
      size: 'small',
    },
  },
  accordion: {
    icons: {
      collapse: CaretDownFill,
      expand: CaretRightFill,
      color: 'brand',
    },
    border: {
      color: 'status-disabled-light',
    },
  },
} as ThemeType);
