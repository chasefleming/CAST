import { HAS_DELAY_ON_START_TIME } from 'const';
import { formatDistance } from 'date-fns';
import { stateToHTML } from 'draft-js-export-html';
import { stateFromHTML } from 'draft-js-import-html';
import { customAlphabet } from 'nanoid';

const nanoid = customAlphabet('1234567890abcdef', 10);

export const generateSlug = nanoid;

export const parseDateFromServer = (endTime) => {
  const dateTime = new Date(endTime);
  const end = dateTime.getTime();
  const now = Date.now();
  const diffFromNow = end - now;
  const diffDays = Math.ceil(Math.abs(diffFromNow) / (1000 * 60 * 60 * 24));
  const diffDuration = formatDistance(now, end); // for the proposalCardFooter
  return {
    date: dateTime,
    diffFromNow,
    diffDays,
    diffDuration,
  };
};

// returns date & time in one object
export const parseDateToServer = (date, time) => {
  const day = new Date(date);
  const hours = new Date(time);
  day.setHours(hours.getHours());
  day.setMinutes(hours.getMinutes());
  return day;
};

export const checkResponse = async (response) => {
  // The Promise returned from fetch() won't reject on HTTP error status even if the response is an HTTP 404 or 500.
  // Instead, as soon as the server responds with headers, the Promise will resolve normally
  // (with the ok property of the response set to false if the response isn't in the range 200–299),
  // and it will only reject on network failure or if anything prevented the request from completing.
  if (!response.ok) {
    const { status, statusText, url } = response;
    const { error } = response.json ? await response.json() : {};
    throw new Error(
      JSON.stringify({ status, statusText: error || statusText, url })
    );
  }
  return response.json();
};

export const isEmptyArray = (array) =>
  Array.isArray(array) && array.length === 0;

export function debounce(e, waitingTime = 300) {
  let timer;
  return (...i) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      e.apply(this, i);
    }, waitingTime);
  };
}
// for some reason, in emulator fcl this signature is nested two levels
// deep but on testnet fcl this is only nested one level deep
export const getSig = (sigArr) =>
  sigArr[0]?.signature?.signature ?? sigArr[0]?.signature;

export const getCompositeSigs = (sigArr) => {
  if (sigArr[0]?.signature?.signature) {
    return [sigArr[0].signature];
  }

  if (typeof sigArr === 'string' && sigArr.includes('Declined:')) {
    return null;
  }
  return sigArr;
};

export function getReducedImg(image, newImageWidth = 150, fileName) {
  const canvas = document.createElement('canvas');
  // if image is bigger then scale
  const scale =
    image.width > newImageWidth ? (newImageWidth / image.width).toFixed(2) : 1;

  canvas.width = image.width * scale;
  canvas.height = image.height * scale;
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.drawImage(
      image,
      0,
      0,
      image.width,
      image.height,
      0,
      0,
      canvas.width,
      canvas.height
    );
  }
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blobImage) => {
        if (blobImage) {
          const blobImageNoType = blobImage;
          //A Blob() is almost a File() - it's just missing the two properties below
          blobImageNoType.lastModifiedDate = new Date();
          blobImageNoType.name = fileName;
          const blobAsFile = blobImageNoType;
          return resolve({ imageFile: blobAsFile });
        }
        reject({ error: 'Error while using blob.' });
      },
      'image/jpeg',
      1
    );
  });
}

export const getProposalType = (choices) => {
  if (
    choices.length === 2 &&
    choices.every(
      (choice) =>
        choice.choiceImgUrl !== undefined && choice.choiceImgUrl !== null
    )
  ) {
    return 'image';
  }
  return 'text-based';
};

// Note: Does not currently return children
export const parseHTML = (body, tag, all) => {
  const parser = new DOMParser();
  const bodyDoc = parser.parseFromString(body, 'text/html');
  const elsFound = bodyDoc.getElementsByTagName(tag);

  if (all) {
    const elArr = Array.from(elsFound);
    if (elArr.length === 0) return {};
    return elArr.map((el) =>
      el.getAttributeNames().reduce((acc, attr) => {
        acc[attr] = el.getAttribute(attr);
        return acc;
      }, {})
    );
  } else {
    const firstEl = Array.from(elsFound)[0];
    if (!firstEl) return {};
    return firstEl.getAttributeNames().reduce((acc, attr) => {
      acc[attr] = firstEl.getAttribute(attr);
      return acc;
    }, {});
  }
};
export const customDraftToHTML = (content) => {
  const options = {
    blockRenderers: {
      'image-caption-block': (block) => {
        return (
          "<p style='font-size: 12px; color: #757575;' class='image-caption'>" +
          block.getText() +
          '</p>'
        );
      },
    },
    entityStyleFn: (entity) => {
      const entityType = entity.get('type').toLowerCase();
      if (entityType === 'link') {
        const data = entity.getData();
        return {
          element: 'a',
          attributes: {
            href: data.url,
            target: '_blank',
            rel: 'noopener noreferrer',
          },
        };
      }
    },
  };
  return stateToHTML(content, options);
};

export const customHTMLtoDraft = (html) => {
  const options = {
    customBlockFn: (element) => {
      if (element.tagName === 'P' && element.className === 'image-caption') {
        return { type: 'image-caption-block' };
      }
    },
  };
  return stateFromHTML(html, options);
};
export const isValidAddress = (addr) => /0[x,X][a-zA-Z0-9]{16}$/gim.test(addr);

export const wait = async (milliSeconds = 5000) =>
  new Promise((resolve) => {
    setTimeout(() => {
      resolve(true);
    }, milliSeconds);
  });
// converts kebab case to regular string
const REVERSE_REGEX = /-[a-zA-Z]/g;

export const kebabToString = (str = '') => {
  const updated = str.replace(REVERSE_REGEX, function (match) {
    return ' ' + match.slice(1);
  });
  return updated.charAt(0).toUpperCase() + updated.slice(1);
};

export const getPaginationFromPages = (pages) => {
  if (!pages) {
    return [];
  }
  const lastPage = [...pages].pop();
  return [
    lastPage?.start ?? 0,
    lastPage?.count ?? 0,
    lastPage?.totalRecords ?? 0,
    lastPage?.next ?? -1,
  ];
};

export const getPagination = (data, countParam) => {
  const [start = 0, count = countParam, totalRecords = 0, next = -1] =
    data?.pageParam ?? getPaginationFromPages(data?.pages);
  return {
    count,
    next,
    start,
    totalRecords,
  };
};

export const getPlainData = (data) => {
  return data?.pages?.reduce(
    (prev, current) => (current.data ? [...prev, ...current.data] : prev),
    []
  );
};
export const validateLength = (string, MaxLength) =>
  string?.length <= MaxLength;

// validates if date made of startTime and startDate is 1 hour ahead of time
export const isStartTimeValid = (startTime, startDate) => {
  const dateNow = new Date();

  const startDateAndTime = new Date(startDate);

  startDateAndTime.setHours(startTime.getHours(), startTime.getMinutes());

  const dif = (startDateAndTime - dateNow) / (60 * 60 * 1000);

  return HAS_DELAY_ON_START_TIME ? dif > 1 : true;
};

export const formatTime = (date) => {
  let hours = date.getHours();
  let minutes = date.getMinutes();
  const ampm = hours >= 12 ? 'pm' : 'am';
  hours = hours % 12;
  hours = hours ? hours : 12; // the hour '0' should be '12'
  minutes = minutes < 10 ? '0' + minutes : minutes;
  return hours + ':' + minutes + ' ' + ampm;
};

export const setDefaultValue = (field, fallbackValue) => {
  if (field === undefined || field === '') {
    return fallbackValue;
  }
  return field;
};

export function truncateAddress(str, initial = 3, tail = 10) {
  return (
    str.substr(0, initial) +
    '...' +
    str.substr((tail - str.length) * -1, str.length)
  );
}
