# 픽셀 테트리스 리눅스 설치

터미널에서 `uname -m`을 실행해 CPU를 확인하세요. `x86_64`는 **amd64**, `aarch64`는 **arm64** 파일을 사용합니다.

## Ubuntu와 Debian

다운로드한 파일이 있는 폴더에서 다음 명령을 실행하세요. ARM64에서는 파일명의 `amd64`를 `arm64`로 바꾸세요.

```sh
sudo apt install ./pixel-tetris_0.1.0_amd64.deb
pixel-tetris
```

삭제하려면 `sudo apt remove pixel-tetris`를 실행하세요.

## 기타 리눅스 배포판

CPU에 맞는 압축 파일을 풀고 바로 실행할 수 있습니다. 아래 예시는 x86-64 기준입니다.

```sh
tar -xzf pixel-tetris-linux-amd64.tar.gz
cd pixel-tetris-linux-amd64
./pixel-tetris
```

## 조작과 소리

터미널을 47열 × 24행 이상으로 열고 **게임 시작** 버튼을 클릭하거나 Enter를 누르세요. 방향키로 이동과 회전, 스페이스로 즉시 낙하, P로 일시정지, R로 재시작, Q로 종료합니다.

효과음에는 오디오 환경에 맞는 재생 도구 하나가 필요합니다. PipeWire는 `pw-play`, PulseAudio는 `paplay`, ALSA는 `aplay`를 사용합니다. Ubuntu와 Debian에서 이 도구들은 각각 `pipewire-bin`, `pulseaudio-utils`, `alsa-utils` 패키지에 들어 있습니다. 게임 중 M으로 효과음을 켜거나 끌 수 있습니다.
