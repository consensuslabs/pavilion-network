import React, { FC, useState } from 'react';
import {
  Anchor,
  Box,
  Button,
  Form,
  FormField,
  TextArea,
  TextInput,
  Text,
} from 'grommet';
import { useDispatch, useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import { validators } from '../../../helpers/validators';
import { updatePostAction } from '../../../store/actions/post.action';
import { AppState } from '../../../store/reducers/root.reducer';
import VideoSettingsTheme from './video-settings-theme';
import Switch from '../../../components/utils/Switch';
import { EditVideoPayload } from '../../../store/types/post.types';
import {
  VIDEO_TITLE_MAX_LENGTH,
  VIDEO_DESCRIPTION_MAX_LENGTH,
} from '../../../helpers/constants';

interface VideoSettingsProps {
  setShowSettings: (settings: boolean) => void;
  setShowDeleteConfirmation: (confirmation: boolean) => void;
}

const VideoSettings: FC<VideoSettingsProps> = ({
  setShowSettings,
  setShowDeleteConfirmation,
}) => {
  const dispatch = useDispatch();
  const { t } = useTranslation();
  const {
    post: {
      postDetail: { data },
    },
  } = useSelector((state: AppState) => state);

  const [publicVisibility, setPublicVisibility] = useState(data?.public);
  const [value, setValue] = useState({
    title: data?.title,
    description: data?.description,
  });

  const onSubmit = () => {
    if (data && value.title) {
      const request: EditVideoPayload = {
        postId: data.id,
        title: value.title,
        description: value.description,
        public: publicVisibility,
      };

      dispatch(updatePostAction(request));
      setShowSettings(false);
    }
  };
  const isVideoSettingsChanged = (): boolean => {
    const isTitleChanged: boolean = value.title !== data?.title;
    const isDescriptionChanged: boolean =
      value.description !== data?.description;
    const isPublicChanged: boolean = publicVisibility !== data?.public;
    const isChanged: boolean =
      isTitleChanged || isDescriptionChanged || isPublicChanged;
    return isChanged;
  };

  return (
    <Box pad="medium" height={{ min: '100vh' }}>
      <VideoSettingsTheme>
        <Form
          value={value}
          onChange={(nextValue) => setValue(nextValue)}
          onSubmit={() => {
            onSubmit();
          }}
        >
          <FormField
            name="title"
            htmlFor="videoName"
            label={t('VideoSettings.videoName')}
            validate={[
              validators.required(t('VideoSettings.videoName')),
              validators.maxLength(VIDEO_TITLE_MAX_LENGTH),
            ]}
          >
            <TextInput
              id="videoName"
              name="title"
              autoFocus
              plain="full"
              type="text"
              maxLength={VIDEO_TITLE_MAX_LENGTH}
              onChange={(event) => {
                setValue({
                  ...value,
                  title: event.target.value,
                });
              }}
              placeholder={t('VideoSettings.videoNamePlaceholder')}
            />
          </FormField>
          <FormField
            name="description"
            htmlFor="videoDescription"
            flex={{ shrink: 0 }}
            label={t('VideoSettings.videoDescription')}
            validate={[validators.maxLength(VIDEO_DESCRIPTION_MAX_LENGTH)]}
          >
            <TextArea
              id="videoDescription"
              placeholder={t('VideoSettings.videoDescriptionPlaceholder')}
              name="description"
              value={value.description}
              maxLength={VIDEO_DESCRIPTION_MAX_LENGTH}
              onChange={(event) => {
                setValue({
                  ...value,
                  description: event.target.value,
                });
              }}
            />
            <Text size="small" color="dark-6" alignSelf="end">
              {value.description ? value.description.length : 0} /{' '}
              {VIDEO_DESCRIPTION_MAX_LENGTH}
            </Text>
          </FormField>
          <Box
            border={{ color: 'status-disabled-light' }}
            round="medium"
            pad="medium"
            gap="medium"
          >
            <Box gap="small">
              <Switch
                label={t('common.public')}
                value={publicVisibility}
                onChange={(checked) => {
                  setPublicVisibility(checked);
                }}
              />
            </Box>
          </Box>
          <Box margin={{ vertical: 'auto' }} />
          <Box gap="small" margin={{ top: '20px' }}>
            <Box direction="row" gap="small">
              <Button
                type="button"
                label={t('common.close')}
                onClick={() => setShowSettings(false)}
                fill="horizontal"
              />
              <Button
                type="submit"
                disabled={!isVideoSettingsChanged()}
                primary
                fill="horizontal"
                label={t('VideoSettings.save')}
              />
            </Box>
            <Box align="center" flex={{ shrink: 0 }} margin={{ top: 'medium' }}>
              <Anchor
                weight="400"
                onClick={() => setShowDeleteConfirmation(true)}
                label={t('VideoSettings.deleteVideo')}
                size="16px"
              />
            </Box>
          </Box>
        </Form>
      </VideoSettingsTheme>
    </Box>
  );
};

export default VideoSettings;
