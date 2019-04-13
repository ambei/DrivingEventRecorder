import {action, computed, configure, observable, runInAction} from "mobx";
import {notification} from 'antd';
import Axios from "../utils/axios"
import backendConfig from "../config/backendConfig";
import dayjs from "dayjs";

configure({enforceActions: "always"});

export default class VideoBasedRecorderStore {
  constructor(rootStore) {
    this.rootStore = rootStore;
  }

  @observable videoList = null;

  @action fetchVideoList = () => {
    Axios.ajax({
      url: backendConfig.videoListApi,
      method: "GET"
    }).then((res) => {
      runInAction(() => {
        this.videoList = res.data || [];
      })
    })
  };

  @observable videoProps = {
    name: null,
    baseTime: null,
    playbackTime: null,
    isFrozen: false,
  };

  @action updateVideoProp = videoProp => {
    this.videoProps[videoProp.key] = videoProp.value;
  };

  @action loadVideo = () => {
    const {name, baseTime} = this.videoProps;
    if (!name || !baseTime) {
      notification.error({
        message: "ValueError",
        description: "Invalid video name or time!"
      });
      return;
    }
    runInAction(() => {
      this.playerProps.url = `${backendConfig.backendVideoURL}/${this.videoProps.name}`;
      this.videoProps.isFrozen = true;
    });
  };

  @action releaseVideo = () => {
    runInAction(() => {
      this.videoProps.isFrozen = false;
      this.videoProps.playbackTime = null;
      this.playerProps.url = null;
      this.playerProps.playbackRate = 1.0;
      this.playerVerticalFlip = false;
      this.playerHorizontalFlip = false;
    });
  };

  @observable playerVerticalFlip = false;
  @observable playerHorizontalFlip = false;

  @action changePlayerFlip = (flipType, flipValue) => {
    if (!flipValue) flipValue = !this[flipType];
    this[flipType] = flipValue;
  };

  @computed get playerFlipStyle () {
    return {
      transform: `scale(${this.playerHorizontalFlip ? -1 : 1}, ${this.playerVerticalFlip ? -1 : 1})`,
      backgroundColor: "#000"
    }
  }

  @action updatePlaybackRate = value => {
    const playbackRate = parseFloat(value);
    if (isNaN(playbackRate) || playbackRate < 0.1 || playbackRate > 5.0) {
      notification.error({
        message: "Value Error",
        description: `Invalid playback rate : ${value}. It should be a number between 0.1 and 5.0.`
      });
      this.playerProps.playbackRate = 1.0;
    } else {
      this.playerProps.playbackRate = playbackRate;
    }
  };

  @action switchPlaying = value => {
    this.playerProps.playing = value === undefined ? !this.playerProps.playing : value;
  };

  @observable lastPlayingState = null;
  @action switchPlayingWithModalVisible = modalVisible => {
    if (modalVisible) {
      this.lastPlayingState = this.playerProps.playing;
      this.switchPlaying(false);
    } else {
      this.switchPlaying(this.lastPlayingState);
    }
  };

  @computed get realTime() {
    const {baseTime, playbackTime} = this.videoProps;
    return baseTime && playbackTime ? dayjs(baseTime).add(playbackTime, 's') : null;
  }

  @computed get realTimeString() {
    return this.realTime ? this.realTime.format("YY-MM-DD HH:mm:ss") : "No value";
  }

  @observable
  playerProps = {
    url: null,
    playing: false,
    loop: false,
    controls: true,
    volume: 1.0,
    muted: false,
    playbackRate: 1,
    height: "40vh",
    style: {margin: "auto"},
    progressInterval: 1000,
    pip: false,
    onReady: () => {
    },
    onStart: () => {
    },
    onPlay: () => {
      this.switchPlaying(true);
    },
    onProgress: (e) => {
      this.updateVideoProp({
        key: 'playbackTime',
        value: e.playedSeconds
      });
      // console.log(e)
    },
    onDuration: (duration) => {
      notification.open({
        message: `Video Loaded (Duration: ${duration.toFixed()}s)`,
        description: `Start playing video to record events.`
      })
    },
    onPause: (e) => {
      runInAction(() => {
        this.updateVideoProp({
          key: 'playbackTime',
          value: e.target.currentTime
        });
        this.switchPlaying(false);
      });
    },
    onEnded: () => {
    },
    onError: (err) => {
      console.error("Error: " + JSON.stringify(err))
    },
    onSeek: (playbackTime) => {
      this.updateVideoProp({
        key: 'playbackTime',
        value: playbackTime
      });
    }
  }

}